package container

import (
	"context"
	"fmt"

	"github.com/yudai/gotty/backend/localcommand"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container/backend"
	"github.com/wrfly/container-web-tty/types"
)

type Cli interface {
	GetInfo(ID string) types.Container
	List(context.Context) []types.Container
	BashExist(ctx context.Context, containerID string) bool
	ShExist(ctx context.Context, containerID string) bool
}

func NewCliBackend(conf config.BackendConfig) (cli Cli, factory *localcommand.Factory, err error) {
	args := []string{}

	switch conf.Type {
	case "docker":
		cli, args, err = backend.NewDockerCli(conf.Docker)
	default:
		err = fmt.Errorf("unknown backend type %s", conf.Type)
	}

	if err != nil {
		return
	}

	args = append(args, conf.ExtraArgs...)

	backendOptions := &localcommand.Options{
		CloseSignal:  1,
		CloseTimeout: -1,
	}
	factory, err = localcommand.NewFactory(args[0], args[1:], backendOptions)
	if err != nil {
		return
	}

	return
}
