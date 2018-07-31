package main

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/yudai/gotty/utils"
	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container"
	"github.com/wrfly/container-web-tty/proxy"
	"github.com/wrfly/container-web-tty/route"
)

func run(c *cli.Context, conf config.Config) {
	appOptions := &route.Options{
		Control: conf.Control,
		Port:    fmt.Sprintf("%d", conf.Server.Port),
		Address: conf.Server.Addr,
		Timeout: conf.Server.IdleTime,
	}
	if err := utils.ApplyDefaultValues(appOptions); err != nil {
		logrus.Fatal(err)
	}

	containerCli, err := container.NewCliBackend(conf.Backend)
	if err != nil {
		logrus.Fatalf("Create backend client error: %s", err)
	}

	srv, err := route.New(containerCli, appOptions)
	if err != nil {
		logrus.Fatalf("Create server error: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	gCtx, gCancel := context.WithCancel(context.Background())

	errs := make(chan error, 2)
	go func() {
		errs <- srv.Run(ctx, route.WithGracefullContext(gCtx))
	}()

	if conf.Server.GrpcPort > 0 {
		grpcServer := proxy.New(conf.Server.GrpcPort, containerCli)
		go func() {
			errs <- grpcServer.Run(ctx)
		}()
	}

	err = waitSignals(errs, cancel, gCancel)

	if err != nil && err != context.Canceled {
		logrus.Fatalf("Server exist with error: %s", err)
	}
}
