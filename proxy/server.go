package proxy

import (
	"context"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/wrfly/container-web-tty/container"
	pb "github.com/wrfly/container-web-tty/proxy/grpc"
	"github.com/wrfly/container-web-tty/types"
	"github.com/wrfly/container-web-tty/util"
)

const (
	rpcCanceld = "rpc error: code = Canceled desc = context canceled"
)

type containerService struct {
	cli  container.Cli
	auth string
}

func newContainerService(cli container.Cli, auth string) pb.ContainerServerServer {
	return &containerService{
		cli:  cli,
		auth: auth,
	}
}

func (svc *containerService) checkAuth(auth string) error {
	if auth != svc.auth {
		return fmt.Errorf("auth failed")
	}
	return nil
}

func (svc *containerService) wrapContainer(cs ...types.Container) []*pb.Container {
	pbContainers := make([]*pb.Container, 0, len(cs))
	for _, c := range cs {
		pbContainers = append(pbContainers, util.ConvertTpContainer(c))
	}
	return pbContainers
}

func checkNil(x interface{}) error {
	if x == nil {
		return fmt.Errorf("nil pointer")
	}
	return nil
}

func (svc *containerService) Ping(ctx context.Context, e *pb.Empty) (*pb.Pong, error) {
	if err := checkNil(e); err != nil {
		return nil, err
	}
	if err := svc.checkAuth(e.Auth); err != nil {
		return nil, err
	}

	return &pb.Pong{
		Msg: "pong",
	}, nil
}

func (svc *containerService) GetInfo(ctx context.Context, cid *pb.ContainerID) (*pb.Container, error) {
	if err := checkNil(cid); err != nil {
		return nil, err
	}

	if err := svc.checkAuth(cid.Auth); err != nil {
		return nil, err
	}

	c := svc.wrapContainer(svc.cli.GetInfo(ctx, cid.Id))[0]
	logrus.Debugf("get info of container: %s (%s)", c.Id, c.Shell)
	return c, nil
}

func (svc *containerService) List(ctx context.Context, e *pb.Empty) (*pb.Containers, error) {
	if err := checkNil(e); err != nil {
		return nil, err
	}

	if err := svc.checkAuth(e.Auth); err != nil {
		return nil, err
	}

	return &pb.Containers{
		Cs: svc.wrapContainer(svc.cli.List(ctx)...),
	}, nil
}

func (svc *containerService) Start(ctx context.Context, cid *pb.ContainerID) (*pb.Err, error) {
	if err := checkNil(cid); err != nil {
		return nil, err
	}

	if err := svc.checkAuth(cid.Auth); err != nil {
		return nil, err
	}

	logrus.Debugf("start container: %s", cid.Id)
	err := svc.cli.Start(ctx, cid.Id)
	if err == nil {
		return nil, nil
	}
	return &pb.Err{
		Err: err.Error(),
	}, nil
}

func (svc *containerService) Stop(ctx context.Context, cid *pb.ContainerID) (*pb.Err, error) {
	if err := checkNil(cid); err != nil {
		return nil, err
	}

	if err := svc.checkAuth(cid.Auth); err != nil {
		return nil, err
	}

	logrus.Debugf("stop container: %s", cid.Id)
	err := svc.cli.Stop(ctx, cid.Id)
	if err == nil {
		return nil, nil
	}
	return &pb.Err{
		Err: err.Error(),
	}, nil
}

func (svc *containerService) Restart(ctx context.Context, cid *pb.ContainerID) (*pb.Err, error) {
	if err := checkNil(cid); err != nil {
		return nil, err
	}

	if err := svc.checkAuth(cid.Auth); err != nil {
		return nil, err
	}

	logrus.Debugf("restart container: %s", cid.Id)
	err := svc.cli.Restart(ctx, cid.Id)
	if err == nil {
		return nil, nil
	}
	return &pb.Err{
		Err: err.Error(),
	}, nil
}

func (svc *containerService) Exec(stream pb.ContainerServer_ExecServer) error {
	// get the initial command and auth and container info
	execOpts, err := stream.Recv()
	if err != nil {
		return err
	}
	if err := svc.checkAuth(execOpts.Auth); err != nil {
		return err
	}
	if execOpts.C == nil {
		return fmt.Errorf("nil container")
	}
	logrus.Debugf("grpc server exec into container: %s", execOpts.C.Id)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	container := util.ConvertPbContainer(execOpts.C)
	logrus.Debugf("container info: %v", container)
	tty, err := svc.cli.Exec(ctx, container)
	if err != nil {
		logrus.Errorf("grpc server exec error: %s", err)
		return err
	}
	defer tty.Exit()

	go func() {
		for {
			execOpts, err := stream.Recv()
			if err != nil {
				if err.Error() == rpcCanceld || err == io.EOF {
					break
				}
				logrus.Debugf("grpc receive outputs error: %s", err)
				break
			}
			if execOpts == nil {
				continue
			}
			// resize terminal
			if ws := execOpts.Ws; ws != nil {
				logrus.Debugf("resize window to %dx%d", ws.Width, ws.Height)
				tty.ResizeTerminal(int(ws.Width), int(ws.Height))
				continue
			}
			// logrus.Debugf("tty write: %s", execOpts.Cmd.In)
			_, err = tty.Write(execOpts.Cmd.In)
			if err == io.EOF {
				continue
			}
			if err != nil {
				logrus.Debugf("tty write error: %s", err)
				break
			}
		}
		logrus.Debugf("grpc server receive done, break")
	}()

	bs := make([]byte, 1024)
	for {
		n, err := tty.Read(bs)
		if err == io.EOF {
			break
		}
		// logrus.Debugf("tty read: %s", bs[:n])
		err = stream.Send(&pb.ExecOptions{
			Cmd: &pb.Io{
				Out: bs[:n],
			},
		})
		if err == io.EOF {
			continue
		}
		if err != nil {
			if err.Error() != rpcCanceld {
				logrus.Debugf("grpc send command error: %s", err)
			}
			break
		}
	}
	logrus.Debugf("tty read done, break")

	logrus.Debugf("grpc exec done")
	return nil
}
