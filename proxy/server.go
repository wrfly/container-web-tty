package proxy

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/wrfly/container-web-tty/container"
	pb "github.com/wrfly/container-web-tty/proxy/grpc"
	"github.com/wrfly/container-web-tty/types"
)

type containerService struct {
	cli container.Cli
}

func newContainerService(cli container.Cli) pb.ContainerServerServer {
	return &containerService{
		cli: cli,
	}
}

func (svc *containerService) wrapContainer(cs ...types.Container) []*pb.Container {
	pbContainers := make([]*pb.Container, 0, len(cs))
	for _, c := range cs {
		pbContainers = append(pbContainers, &pb.Container{
			Id:            c.ID,
			Name:          c.Name,
			Image:         c.Image,
			Command:       c.Command,
			State:         c.State,
			Status:        c.Status,
			Ips:           c.IPs,
			Shell:         c.Shell,
			PodName:       c.PodName,
			ContainerName: c.ContainerName,
			Namespace:     c.Namespace,
			RunningNode:   c.RunningNode,
			LocServer:     c.LocServer,
		})
	}
	return pbContainers
}

func (svc *containerService) wrapPbContainer(c *pb.Container) types.Container {
	return types.Container{
		ID:            c.Id,
		Name:          c.Name,
		Image:         c.Image,
		Command:       c.Command,
		State:         c.State,
		Status:        c.Status,
		IPs:           c.Ips,
		Shell:         c.Shell,
		PodName:       c.PodName,
		ContainerName: c.ContainerName,
		Namespace:     c.Namespace,
		RunningNode:   c.RunningNode,
		LocServer:     c.LocServer,
	}
}

func (svc *containerService) GetInfo(ctx context.Context, cid *pb.ContainerID) (*pb.Container, error) {
	c := svc.wrapContainer(svc.cli.GetInfo(ctx, cid.Id))[0]
	logrus.Debugf("get info of container: %s (%s)", c.Id, c.Shell)
	return c, nil
}

func (svc *containerService) List(ctx context.Context, _ *pb.Empty) (*pb.Containers, error) {
	return &pb.Containers{
		Cs: svc.wrapContainer(svc.cli.List(ctx)...),
	}, nil
}

func (svc *containerService) Start(ctx context.Context, cid *pb.ContainerID) (*pb.Err, error) {
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
	// get the initial command
	execOpts, err := stream.Recv()
	if err != nil {
		return err
	}
	if execOpts.C == nil {
		return fmt.Errorf("nil container")
	}
	logrus.Debugf("grpc server exec into container: %s", execOpts.C.Id)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	containerInfo := svc.wrapPbContainer(execOpts.C)
	logrus.Debugf("container info: %v", containerInfo)
	tty, err := svc.cli.Exec(ctx, containerInfo)
	if err != nil {
		logrus.Errorf("grpc server exec error: %s", err)
		return err
	}
	defer tty.Exit()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer logrus.Debugf("grpc server receive done, break")
		for {
			execOpts, err := stream.Recv()
			if err == io.EOF {
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
			logrus.Debugf("tty write: %s", execOpts.Cmd.In)
			_, err = tty.Write(execOpts.Cmd.In)
			if err == io.EOF {
				continue
			}
			if err != nil {
				logrus.Debugf("grpc exec got error: %s", err)
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer logrus.Debugf("tty read done, break")
		bs := make([]byte, 1024)
		for {
			n, err := tty.Read(bs)
			if err == io.EOF {
				break
			}
			logrus.Debugf("tty read: %s", bs[:n])
			err = stream.Send(&pb.ExecOptions{
				Cmd: &pb.Io{
					Out: bs[:n],
				},
			})
			if err == io.EOF {
				continue
			}
			if err != nil {
				logrus.Debugf("grpc exec got error: %s", err)
				break
			}
		}
	}()

	wg.Wait()
	logrus.Debugf("grpc exec done")
	return nil
}
