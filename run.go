package main

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/wrfly/ecp"
	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container"
	"github.com/wrfly/container-web-tty/proxy"
	"github.com/wrfly/container-web-tty/route"
	"github.com/wrfly/container-web-tty/util"
)

func run(c *cli.Context, conf config.Config) {
	srvOptions := conf.Server

	if len(conf.Backend.GRPC.Servers) > 0 {
		srvOptions.ShowLocation = true
	}
	if err := ecp.Default(&srvOptions); err != nil {
		logrus.Fatal(err)
	}

	if srvOptions.GrpcPort <= 0 && srvOptions.Port <= 0 {
		logrus.Fatal("bad config, no port listenning")
	}

	containerCli, err := container.NewCliBackend(conf.Backend)
	if err != nil {
		logrus.Fatalf("Create backend client error: %s", err)
	}
	defer containerCli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	gCtx, gCancel := context.WithCancel(ctx)
	errs := make(chan error, 2)

	// run HTTP server if port > 0
	if srvOptions.Port > 0 {
		go func() {
			srv, err := route.New(containerCli, srvOptions)
			if err != nil {
				logrus.Fatalf("Create server error: %s", err)
			}
			errs <- srv.Run(ctx, route.WithGracefullContext(gCtx))
		}()
	}

	// run grpc server if grpc-port > 0
	if srvOptions.GrpcPort > 0 {
		go func() {
			grpcServer := proxy.New(conf.Backend.GRPC.Auth,
				srvOptions.GrpcPort, containerCli)
			errs <- grpcServer.Run(ctx, gCtx)
		}()
	}

	err = util.WaitSignals(errs, cancel, gCancel)
	if err != nil && err != context.Canceled {
		logrus.Fatalf("Server closed with error: %s", err)
	}
	logrus.Info("Server closed")
}
