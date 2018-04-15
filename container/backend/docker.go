package backend

import (
	"context"

	apiTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/types"
)

type DockerCli struct {
	cli *client.Client
}

func NewDockerCli(conf config.DockerConfig) (*DockerCli, []string, error) {
	host := conf.DockerHost
	if host[:1] == "/" {
		host = "unix://" + host
	} else {
		host = "tcp://" + host
	}
	version := "v1.24"
	UA := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient(host, version, nil, UA)
	if err != nil {
		logrus.Errorf("create new docker client error: %s", err)
		return nil, nil, err
	}
	return &DockerCli{
		cli: cli,
	}, []string{conf.DockerPath, "exec", "-ti"}, nil
}

func getContainerIP(networkSettings *apiTypes.SummaryNetworkSettings) []string {
	ips := []string{}

	if networkSettings == nil {
		return ips
	}

	for net := range networkSettings.Networks {
		ips = append(ips, networkSettings.Networks[net].IPAddress)
	}

	return ips
}

func (docker DockerCli) GetInfo(ID string) types.Container {
	return types.Container{}
}

func (docker DockerCli) List(ctx context.Context) []types.Container {
	cs, err := docker.cli.ContainerList(ctx, apiTypes.ContainerListOptions{})
	if err != nil {
		logrus.Errorf("list containers eror: %s", err)
		return nil
	}
	containers := []types.Container{}
	for _, container := range cs {
		containers = append(containers, types.Container{
			ID:      container.ID,
			Name:    container.Names[0][1:],
			Image:   container.Image,
			Command: container.Command,
			IPs:     getContainerIP(container.NetworkSettings),
			Status:  container.Status,
			State:   container.State,
		})
	}
	return containers
}

func (docker DockerCli) BashExist(ctx context.Context, cid string) bool {
	_, err := docker.cli.ContainerStatPath(ctx, cid, "/bin/bash")
	if err != nil {
		return false
	}
	return true
}
