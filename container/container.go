package container

import (
	"context"
	"fmt"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container/docker"
	"github.com/wrfly/container-web-tty/types"
)

type Cli interface {
	// GetInfo of a container
	GetInfo(ctx context.Context, containerID string) types.Container
	// List all containers
	List(context.Context) []types.Container
	// GetShell returns the shell name, bash>ash>sh
	GetShell(ctx context.Context, containerID string) string
	Start(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Restart(ctx context.Context, containerID string) error
	// exec into container
	Exec(ctx context.Context, container types.Container) (types.TTY, error)
}

func NewCliBackend(conf config.BackendConfig) (cli Cli, err error) {
	switch conf.Type {
	case "docker":
		cli, err = docker.NewCli(conf.Docker, conf.ExtraArgs)
	// case "kube":
	// cli, args, err = kube.NewCli(conf.Kube)
	default:
		err = fmt.Errorf("unknown backend type %s", conf.Type)
	}

	return
}
