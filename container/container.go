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
}

func NewCli(typ string, Opts interface{}) (Cli, error) {
	switch typ {
	case "docker":
		return backend.NewDockerCli(Opts.(config.DockerConfig))
	default:
		return nil, fmt.Errorf("unknown backend type %s", typ)
	}
}
