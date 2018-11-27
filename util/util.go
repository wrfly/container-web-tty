package util

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	pb "github.com/wrfly/container-web-tty/proxy/pb"
	"github.com/wrfly/container-web-tty/types"
)

// ConvertTpContainer *pb.Container -> types.Container
func ConvertPbContainer(c *pb.Container) types.Container {
	return types.Container{
		ID:            c.Id,
		Name:          c.Name,
		Image:         c.Image,
		Command:       c.Command,
		State:         c.State,
		Status:        c.Status,
		IPs:           c.Ips,
		Shell:         c.Shell,
		PodName:       c.PodName,
		ContainerName: c.ContainerName,
		Namespace:     c.Namespace,
		RunningNode:   c.RunningNode,
		LocServer:     c.LocServer,
		ExecCMD:       c.ExecCMD,
	}
}

// ConvertTpContainer types.Container -> *pb.Container
func ConvertTpContainer(c types.Container) *pb.Container {
	return &pb.Container{
		Id:            c.ID,
		Name:          c.Name,
		Image:         c.Image,
		Command:       c.Command,
		State:         c.State,
		Status:        c.Status,
		Ips:           c.IPs,
		Shell:         c.Shell,
		PodName:       c.PodName,
		ContainerName: c.ContainerName,
		Namespace:     c.Namespace,
		RunningNode:   c.RunningNode,
		LocServer:     c.LocServer,
		ExecCMD:       c.ExecCMD,
	}
}

func HomeDIR() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func DockerCliPath() string {
	if runtime.GOOS == `darwin` {
		return "/usr/local/bin/docker"
	}
	return "/usr/bin/docker"
}

func KubeConfigPath() string {
	home := HomeDIR()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".kube", "config")
}

func EnvVars(e string) []string {
	e = strings.ToUpper(e)
	return []string{"WEB_TTY_" + strings.Replace(e, "-", "_", -1)}
}

func WaitSignals(errs chan error, cancel context.CancelFunc, gracefullCancel context.CancelFunc) error {
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
			select {
			case err := <-errs:
				return err
			case <-sigChan:
				fmt.Println("Force closing...")
				cancel()
				return nil
			}
		default:
			cancel()
			return nil
		}
	}
}
