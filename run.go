package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/yudai/gotty/utils"
	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container"
	"github.com/wrfly/container-web-tty/route"
)

func run(c *cli.Context, conf config.Config) {
	appOptions := &route.Options{}
	if err := utils.ApplyDefaultValues(appOptions); err != nil {
		log.Fatal(err)
	}

	appOptions.Port = fmt.Sprint(conf.Port)

	containerCli, factory, err := container.NewCliBackend(conf.Backend)
	if err != nil {
		log.Fatalf("create backend client error: %s", err)
	}

	srv, err := route.New(factory, appOptions, containerCli)
	if err != nil {
		log.Fatalf("create server error: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	gCtx, gCancel := context.WithCancel(context.Background())

	errs := make(chan error, 1)
	go func() {
		errs <- srv.Run(ctx, route.WithGracefullContext(gCtx))
	}()

	err = waitSignals(errs, cancel, gCancel)

	if err != nil && err != context.Canceled {
		log.Fatalf("server exist with error: %s", err)
	}
}

func waitSignals(errs chan error, cancel context.CancelFunc, gracefullCancel context.CancelFunc) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(
		sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	select {
	case err := <-errs:
		return err

	case s := <-sigChan:
		switch s {
		case syscall.SIGINT:
			gracefullCancel()
			fmt.Println("C-C to force close")
			select {
			case err := <-errs:
				return err
			case <-sigChan:
				fmt.Println("Force closing...")
				cancel()
				return <-errs
			}
		default:
			cancel()
			return <-errs
		}
	}
}
