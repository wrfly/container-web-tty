package container

import (
	"context"
	"fmt"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container/docker"
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
}

// NewCliBackend returns the client backend
func NewCliBackend(conf config.BackendConfig) (cli Cli, err error) {
	switch conf.Type {
	case "docker":
		cli, err = docker.NewCli(conf.Docker, conf.ExtraArgs)
	case "kube":
		cli, err = kube.NewCli(conf.Kube, conf.ExtraArgs)
	default:
		err = fmt.Errorf("unknown backend type %s", conf.Type)
	}

	return
}
