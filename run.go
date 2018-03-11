package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/gotty/backend/localcommand"
	"github.com/wrfly/container-web-tty/gotty/server"
	"github.com/wrfly/container-web-tty/gotty/utils"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container"
)

func run(c *cli.Context, conf config.Config) {
	appOptions := &server.Options{}
	if err := utils.ApplyDefaultValues(appOptions); err != nil {
		exit(err, 1)
	}
	backendOptions := &localcommand.Options{}
	if err := utils.ApplyDefaultValues(backendOptions); err != nil {
		exit(err, 1)
	}

	appOptions.Port = fmt.Sprint(conf.Port)
	appOptions.Address = "0.0.0.0"
	appOptions.PermitWrite = true

	err := appOptions.Validate()
	if err != nil {
		exit(err, 6)
	}

	hostname, _ := os.Hostname()
	appOptions.TitleVariables = map[string]interface{}{
		"hostname":      hostname,
		"containerName": "",
		"containerID":   "",
	}
	containerCli, cmds, err := container.NewCli(conf.Backend)
	if err != nil {
		exit(err, 3)
	}

	defaultFactory, err := localcommand.NewFactory(cmds[0], cmds[1:], backendOptions)
	if err != nil {
		exit(err, 3)
	}

	srv, err := server.New(defaultFactory, appOptions, containerCli)
	if err != nil {
		exit(err, 3)
	}

	ctx, cancel := context.WithCancel(context.Background())
	gCtx, gCancel := context.WithCancel(context.Background())

	errs := make(chan error, 1)
	go func() {
		errs <- srv.Run(ctx, server.WithGracefullContext(gCtx))
	}()
	err = waitSignals(errs, cancel, gCancel)

	if err != nil && err != context.Canceled {
		fmt.Printf("Error: %s\n", err)
		exit(err, 8)
	}

}

func exit(err error, code int) {
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(code)
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
