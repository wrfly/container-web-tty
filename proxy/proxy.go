package proxy

import (
	"context"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/wrfly/container-web-tty/container"
	pbrpc "github.com/wrfly/container-web-tty/proxy/grpc"
)

// GrpcServer is the grpc server
type GrpcServer interface {
	Run(ctx context.Context) error
}

type grpcServer struct {
	port int
	cli  container.Cli
}

// New proxy grpc server
func New(port int, cli container.Cli) GrpcServer {
	logrus.Infof("New grpc server with port %d", port)
	return &grpcServer{
		port: port,
		cli:  cli,
	}
}

// Run the server
func (gsrv *grpcServer) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", gsrv.port))
	if err != nil {
		return err
	}
	srv := grpc.NewServer()

	cs := newContainerService(gsrv.cli)
	pbrpc.RegisterContainerServerServer(srv, cs)

	// serve
	go func() {
		logrus.Infof("Running grpc server at :%d", gsrv.port)
		if err := srv.Serve(listener); err != nil {
			logrus.Errorf("GRPC API server error: %s", err)
		} else {
			logrus.Infof("GRPC API server stopped: %s", ctx.Err())
		}
	}()

	// shutdown with cancel
	<-ctx.Done()
	srv.GracefulStop()
	return listener.Close()
}
