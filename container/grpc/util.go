package remote

import "github.com/wrfly/container-web-tty/types"
import pb "github.com/wrfly/container-web-tty/proxy/grpc"

func convertToContainer(c *pb.Container) types.Container {
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
	}
}

func convertToPB(c types.Container) *pb.Container {
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
	}
}
