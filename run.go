package main

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/yudai/gotty/utils"
	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container"
	"github.com/wrfly/container-web-tty/route"
)

func run(c *cli.Context, conf config.Config) {
	appOptions := &route.Options{}
	if err := utils.ApplyDefaultValues(appOptions); err != nil {
		logrus.Fatal(err)
	}

	appOptions.Port = fmt.Sprint(conf.Port)

	containerCli, factory, err := container.NewCliBackend(conf.Backend)
	if err != nil {
		logrus.Fatalf("create backend client error: %s", err)
	}

	srv, err := route.New(factory, appOptions, containerCli)
	if err != nil {
		logrus.Fatalf("create server error: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	gCtx, gCancel := context.WithCancel(context.Background())

	errs := make(chan error, 1)
	go func() {
		errs <- srv.Run(ctx, route.WithGracefullContext(gCtx))
	}()

	err = waitSignals(errs, cancel, gCancel)

	if err != nil && err != context.Canceled {
		logrus.Fatalf("server exist with error: %s", err)
	}
}
