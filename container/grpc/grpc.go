package remote

import (
	"context"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/wrfly/container-web-tty/config"
	pb "github.com/wrfly/container-web-tty/proxy/grpc"
	"github.com/wrfly/container-web-tty/types"
	"github.com/wrfly/container-web-tty/util"
)

type grpcCli struct {
	addr, auth string

	conn   *grpc.ClientConn
	client pb.ContainerServerClient
}

func (g *grpcCli) close() error {
	return g.conn.Close()
}

func (g *grpcCli) waitForAvaliable(ctx context.Context) {
	g.conn.WaitForStateChange(ctx, connectivity.TransientFailure)
}

func (g *grpcCli) alive() bool {
	pong, err := g.client.Ping(context.Background(), &pb.Empty{Auth: g.auth})
	if err != nil {
		logrus.Errorf("connot connect to [%s], %s", g.addr, err)
		return false
	}
	logrus.Debugf("pong from %s: %s", g.addr, pong.GetMsg())
	return true
}

func (g *grpcCli) state() connectivity.State {
	return g.conn.GetState()
}

// GrpcCli connect to the remote server
type GrpcCli struct {
	servers    []string
	auth       string
	clients    map[string]grpcCli
	containers *types.Containers
}

// NewCli returns the GrpcCli
func NewCli(conf config.GRPCConfig) (*GrpcCli, error) {
	logrus.Infof("New gRPC client connect to %v with auth [%s]",
		conf.Servers, conf.Auth)
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
			auth:   conf.Auth,
			addr:   serverAddr,
			conn:   conn,
			client: pb.NewContainerServerClient(conn),
		}
	}

	delLists := make([]string, 0, len(gCli.clients))
	for addr, cli := range gCli.clients {
		if cli.alive() {
			continue
		}
		delLists = append(delLists, addr)
	}
	for _, addr := range delLists {
		logrus.Infof("connot ping remote server %s, remove it from clients", addr)
		delete(gCli.clients, addr)
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
	pbContainer, err := remoteClient.client.GetInfo(ctx,
		&pb.ContainerID{Id: cid, Auth: gCli.auth})
	if err != nil {
		logrus.Errorf("grpc get container error: %s", err)
		return types.Container{}
	}
	gCli.containers.SetShell(cid, pbContainer.GetShell())
	return util.ConvertPbContainer(pbContainer)
}

func (gCli GrpcCli) List(ctx context.Context) []types.Container {
	allContainers := make([]types.Container, 0)
	for addr, cli := range gCli.clients {
		if !cli.alive() {
			logrus.Warnf("remote server %s is not ready: %s", addr, cli.state())
			continue
		}
		cs, err := cli.client.List(ctx, &pb.Empty{Auth: gCli.auth})
		if err != nil {
			logrus.Errorf("get container info error: %s", err)
			continue
		}
		containers := make([]types.Container, len(cs.Cs))
		for i, c := range cs.Cs {
			c.LocServer = addr
			containers[i] = util.ConvertPbContainer(c)
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
		pbCID := &pb.ContainerID{
			Id:   containerID,
			Auth: gCli.auth,
		}
		switch action {
		case "start":
			err1, err2 = cli.client.Start(ctx, pbCID)
		case "stop":
			err1, err2 = cli.client.Stop(ctx, pbCID)
		case "restart":
			err1, err2 = cli.client.Restart(ctx, pbCID)
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
	if !cli.alive() {
		return nil, fmt.Errorf("remote server %s is not ready: %s", container.LocServer, cli.state())
	}

	execClient, err := cli.client.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// send container info to server,
	// server will use another cli to exec into the container
	if err := execClient.Send(&pb.ExecOptions{
		C:    util.ConvertTpContainer(container),
		Auth: gCli.auth,
	}); err != nil {
		return nil, err
	}

	// start to read and write using this exec wrapper
	return newExecWrapper(execClient), nil
}

func (gCli GrpcCli) Close() error {
	for addr, cli := range gCli.clients {
		if err := cli.close(); err != nil {
			logrus.Errorf("close %s error: %s", addr, err)
		}
	}
	return nil
}

func (gCli GrpcCli) Logs(ctx context.Context, opts types.LogOptions) (io.ReadCloser, error) {
	logrus.Debugf("get container logs, id: %s", opts.ID)
	info := gCli.containers.Find(opts.ID)
	if info.ID == "" {
		return nil, fmt.Errorf("container not found")
	}

	cli, exist := gCli.clients[info.LocServer]
	if !exist {
		return nil, fmt.Errorf("location server [%s] not found", info.LocServer)
	}
	if !cli.alive() {
		logrus.Error("111")
		return nil, fmt.Errorf("remote server %s is not ready: %s", info.LocServer, cli.state())
	}

	logsClient, err := cli.client.Logs(ctx, &pb.LogOpts{
		Follow: opts.Follow,
		Tail:   opts.Tail,
		C: &pb.ContainerID{
			Id:   info.ID,
			Auth: gCli.auth,
		},
	})
	if err != nil {
		return nil, err
	}

	logrus.Debugf("start to pipe...")
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		defer pr.Close()
		for {
			in, err := logsClient.Recv()
			if err != nil {
				if grpc.ErrorDesc(err) != context.Canceled.Error() {
					logrus.Errorf("logs recv error: %s", err)
				}
				break
			}
			if _, err = pw.Write(in.GetIn()); err != nil {
				logrus.Errorf("logs write to remote error: %s", err)
				break
			}
		}
	}()

	return pr, nil
}
