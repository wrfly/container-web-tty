package container

import (
	"context"
	"fmt"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container/backend"
	"github.com/wrfly/container-web-tty/types"
)

type Cli interface {
	List(context.Context) []types.Container
	BashExist(ctx context.Context, containerID string) bool
}

func NewCli(conf config.BackendConfig) (Cli, []string, error) {
	switch conf.Type {
	case "docker":
		return backend.NewDockerCli(conf.Docker)
	default:
		return nil, nil, fmt.Errorf("unknown backend type %s", conf.Type)
	}
}
