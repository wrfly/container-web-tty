package remote

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/wrfly/container-web-tty/config"
	pb "github.com/wrfly/container-web-tty/proxy/grpc"
	"github.com/wrfly/container-web-tty/types"
	"google.golang.org/grpc"
)

// grpc client, connect to the remote server

type grpcCli struct {
	conn   *grpc.ClientConn
	client *pb.ContainerServerClient
}

type GrpcCli struct {
	servers []string
	auth    string
	clients map[string]grpcCli
}

func NewGrpcCli(conf config.RemoteConfig) (*GrpcCli, error) {
	grpcCli := &GrpcCli{
		servers: conf.Servers,
		auth:    conf.Auth,
		clients: make(map[string]grpcCli, len(conf.Servers)),
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
		grpcCli.clients[serverAddr] = grpcCli{
			conn:   conn,
			client: pb.NewRouteGuideClient(conn),
		}
	}
	return grpcCli, nil
}

func (gcli GrpcCli) GetInfo(ctx context.Context, containerID string) types.Container {

	for _,cli := range gcli.clients{
		cli.client.
	}

	return
}

func (gcli GrpcCli) List(context.Context) []types.Container {

}

func (gcli GrpcCli) Start(ctx context.Context, containerID string) error {

}

func (gcli GrpcCli) Stop(ctx context.Context, containerID string) error {

}

func (gcli GrpcCli) Restart(ctx context.Context, containerID string) error {

}

func (gcli GrpcCli) Exec(ctx context.Context, container types.Container) (types.TTY, error) {

}
