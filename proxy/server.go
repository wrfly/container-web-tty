package proxy

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/wrfly/container-web-tty/container"
	pb "github.com/wrfly/container-web-tty/proxy/grpc"
	"github.com/wrfly/container-web-tty/types"
)

// service containerServer {
//     rpc GetInfo (ConatinerID) returns (Container) {}
//     rpc List (empty) returns (Containers) {}
//     rpc Start (ConatinerID) returns (error) {}
//     rpc Stop (ConatinerID) returns (error) {}
//     rpc Restart (ConatinerID) returns (error) {}
//     rpc Exec(stream Command) returns (stream Command) {}
// }

type containerService struct {
	cli container.Cli
}

func newContainerService(cli container.Cli) *containerService {
	return &containerService{
		cli: cli,
	}
}

func (svc *containerService) wrapContainer(c ...types.Container) []*pb.Container {
	pbContainers := make([]*pb.Container, 0, len(c))
	for _, container := range c {
		pbContainers = append(pbContainers, &pb.Container{
			Id: container.ID,
		})
	}
	return pbContainers
}

func (svc *containerService) wrapPbContainer(c *pb.Container) types.Container {
	return types.Container{}
}

func (svc *containerService) GetInfo(ctx context.Context, cid *pb.ContainerID) (*pb.Container, error) {
	return svc.wrapContainer(svc.cli.GetInfo(ctx, cid.Id))[0], nil
}

func (svc *containerService) List(ctx context.Context, _ *pb.Empty) (*pb.Containers, error) {
	return &pb.Containers{
		Cs: svc.wrapContainer(svc.cli.List(ctx)...),
	}, nil
}

func (svc *containerService) Start(ctx context.Context, cid *pb.ContainerID) (*pb.Err, error) {
	err := svc.cli.Start(ctx, cid.Id)
	if err == nil {
		return nil, nil
	}
	return &pb.Err{
		Err: err.Error(),
	}, nil
}

func (svc *containerService) Stop(ctx context.Context, cid *pb.ContainerID) (*pb.Err, error) {
	err := svc.cli.Stop(ctx, cid.Id)
	if err == nil {
		return nil, nil
	}
	return &pb.Err{
		Err: err.Error(),
	}, nil
}

func (svc *containerService) Restart(ctx context.Context, cid *pb.ContainerID) (*pb.Err, error) {
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tty, err := svc.cli.Exec(ctx, svc.wrapPbContainer(execOpts.C))
	if err != nil {
		return err
	}
	defer tty.Exit()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for {
			execOpts, err := stream.Recv()
			if err == io.EOF {
				break
			}
			_, err = tty.Write(execOpts.Cmd.In)
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		defer wg.Done()
		bs := make([]byte, 2048)
		for {
			n, err := tty.Read(bs)
			if err == io.EOF {
				break
			}
			err = stream.Send(&pb.ExecOptions{
				Cmd: &pb.Io{
					Out: bs[:n],
				},
			})
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
		}
	}()

	wg.Wait()
	return nil
}
