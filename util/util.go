package util

import (
	"os"

	pb "github.com/wrfly/container-web-tty/proxy/grpc"
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
