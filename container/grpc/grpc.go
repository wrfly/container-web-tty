package remote

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/wrfly/container-web-tty/config"
	pb "github.com/wrfly/container-web-tty/proxy/grpc"
	"github.com/wrfly/container-web-tty/types"
)

// grpc client, connect to the remote server

type grpcCli struct {
	conn   *grpc.ClientConn
	client pb.ContainerServerClient
}

type GrpcCli struct {
	servers    []string
	auth       string
	clients    map[string]grpcCli
	containers *types.Containers
}

func NewCli(conf config.GRPCConfig) (*GrpcCli, error) {
	gCli := &GrpcCli{
		servers:    conf.Servers,
		auth:       conf.Auth,
		clients:    make(map[string]grpcCli, len(conf.Servers)),
		containers: new(types.Containers),
	}

	var opts []grpc.DialOption
	// if *tls {
	// 	if *caFile == "" {
	// 		*caFile = testdata.Path("ca.pem")
	// 	}
	// 	creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
	// 	if err != nil {
	// 		log.Fatalf("Failed to create TLS credentials %v", err)
	// 	}
	// 	opts = append(opts, grpc.WithTransportCredentials(creds))
	// }
	opts = append(opts, grpc.WithInsecure())
	for _, serverAddr := range conf.Servers {
		conn, err := grpc.Dial(serverAddr, opts...)
		if err != nil {
			logrus.Errorf("fail to dial: %v", err)
			continue
		}
		gCli.clients[serverAddr] = grpcCli{
			conn:   conn,
			client: pb.NewContainerServerClient(conn),
		}
	}
	return gCli, nil
}

func (gCli GrpcCli) GetInfo(ctx context.Context, cid string) types.Container {
	if gCli.containers.Len() == 0 {
		logrus.Debugf("zero containers, get cid %s", cid)
		gCli.List(ctx)
	}

	container := gCli.containers.Find(cid)
	if container.ID == "" {
		logrus.Errorf("no such container: %s", cid)
		return types.Container{}
	}
	if container.Shell != "" {
		logrus.Debugf("found valid container: %s (%s)", container.ID, container.Shell)
		return container
	}

	remoteAddr := container.LocServer
	remoteClient, exist := gCli.clients[remoteAddr]
	if !exist {
		logrus.Errorf("no remote client: %s", remoteAddr)
		return types.Container{}
	}
	pbContainer, err := remoteClient.client.GetInfo(ctx, &pb.ContainerID{Id: cid})
	if err != nil {
		logrus.Errorf("grpc get container error: %s", err)
		return types.Container{}
	}
	gCli.containers.SetShell(cid, pbContainer.GetShell())
	return convertToContainer(pbContainer)
}

func (gCli GrpcCli) List(ctx context.Context) []types.Container {
	allContainers := make([]types.Container, 0)
	for addr, cli := range gCli.clients {
		cs, err := cli.client.List(ctx, &pb.Empty{})
		if err != nil {
			logrus.Errorf("get container info error: %s", err)
			continue
		}
		containers := make([]types.Container, len(cs.Cs))
		for i, c := range cs.Cs {
			c.LocServer = addr
			containers[i] = convertToContainer(c)
		}
		allContainers = append(allContainers, containers...)
	}

	gCli.containers.Set(allContainers)
	logrus.Debugf("list %d containers", len(allContainers))

	return allContainers
}

func (gCli GrpcCli) containerAction(ctx context.Context, action, containerID string) error {
	info := gCli.containers.Find(containerID)
	if info.ID == "" {
		return fmt.Errorf("container not found")
	}
	if cli, exist := gCli.clients[info.LocServer]; exist {
		var err1 *pb.Err
		var err2 error
		switch action {
		case "start":
			err1, err2 = cli.client.Start(ctx, &pb.ContainerID{
				Id: containerID,
			})
		case "stpp":
			err1, err2 = cli.client.Stop(ctx, &pb.ContainerID{
				Id: containerID,
			})
		case "restart":
			err1, err2 = cli.client.Restart(ctx, &pb.ContainerID{
				Id: containerID,
			})
		default:
			return fmt.Errorf("unknown action: %s", action)
		}

		if err2 != nil {
			return err2
		}
		if err1 != nil {
			return fmt.Errorf(err1.Err)
		}
		return nil
	}
	return fmt.Errorf("location server [%s] not found", info.LocServer)
}

func (gCli GrpcCli) Start(ctx context.Context, containerID string) error {
	return gCli.containerAction(ctx, "start", containerID)
}

func (gCli GrpcCli) Stop(ctx context.Context, containerID string) error {
	return gCli.containerAction(ctx, "stop", containerID)
}

func (gCli GrpcCli) Restart(ctx context.Context, containerID string) error {
	return gCli.containerAction(ctx, "restart", containerID)
}

func (gCli GrpcCli) Exec(ctx context.Context, container types.Container) (types.TTY, error) {
	logrus.Debugf("exec into container: %s (%s)", container.ID, container.Shell)
	if container.ID == "" {
		return nil, fmt.Errorf("container not found")
	}

	cli, exist := gCli.clients[container.LocServer]
	if !exist {
		return nil, fmt.Errorf("location server [%s] not found", container.LocServer)
	}

	execClient, err := cli.client.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// send container info to server, server will use another cli to exec into the container
	if err := execClient.Send(&pb.ExecOptions{C: convertToPB(container)}); err != nil {
		return nil, err
	}

	// start to read and write
	enj := newExecInjector(execClient)

	// // web input
	// go func(){
	// 	buff := make([]byte, 2048)
	// 	enj.ttyIn
	// 	}()

	// // web output
	// go func(){
	// 	buff := make([]byte, 2048)
	// 	enj.ttyOut
	// }()

	return enj, nil
}
