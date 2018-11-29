package container

import (
	"context"
	"fmt"
	"io"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container/docker"
	"github.com/wrfly/container-web-tty/container/grpc"
	"github.com/wrfly/container-web-tty/container/kube"
	"github.com/wrfly/container-web-tty/types"
)

// Cli is a docker backend client
type Cli interface {
	// GetInfo of a container
	GetInfo(ctx context.Context, containerID string) types.Container
	// List all containers
	List(context.Context) []types.Container
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Restart(ctx context.Context, containerID string) error
	// exec into container
	Exec(ctx context.Context, container types.Container) (types.TTY, error)
	// close the connections
	Close() error
	// read logs
	Logs(ctx context.Context, opts types.LogOptions) (io.ReadCloser, error)
}

// NewCliBackend returns the client backend
func NewCliBackend(conf config.BackendConfig) (cli Cli, err error) {
	switch conf.Type {
	case "docker":
		cli, err = docker.NewCli(conf.Docker)
	case "kube":
		cli, err = kube.NewCli(conf.Kube)
	case "grpc":
		cli, err = grpc.NewCli(conf.GRPC)
	default:
		err = fmt.Errorf("unknown backend type %s", conf.Type)
	}

	return
}
