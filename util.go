package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/wrfly/container-web-tty/util"
)

func dockerCliPath() string {
	if runtime.GOOS == `darwin` {
		return "/usr/local/bin/docker"
	}
	return "/usr/bin/docker"
}

func kubeConfigPath() string {
	home := util.HomeDIR()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

func envVars(e string) []string {
	e = strings.ToUpper(e)
	return []string{"WEB_TTY_" + strings.Replace(e, "-", "_", -1)}
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
