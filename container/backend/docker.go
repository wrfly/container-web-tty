package backend

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	apiTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/types"
)

type DockerCli struct {
	cli             *client.Client
	containers      map[string]types.Container
	containersMutex *sync.RWMutex
	listOptions     apiTypes.ContainerListOptions
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

	listOptions, err := buildListOptions(conf.PsOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("build ps options error: %s", err)
	}
	logrus.Debugf("%+v", listOptions)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ping, err := cli.Ping(ctx)
	if err != nil {
		return nil, nil, err
	}
	logrus.Infof("New docker client: OS [%s], API [%s]", ping.OSType, ping.APIVersion)

	return &DockerCli{
		cli:             cli,
		containers:      map[string]types.Container{},
		containersMutex: &sync.RWMutex{},
		listOptions:     listOptions,
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

func (docker DockerCli) GetInfo(ctx context.Context, cid string) types.Container {
	if len(docker.containers) == 0 {
		docker.List(ctx)
	}
	docker.containersMutex.RLock()
	containers := docker.containers
	docker.containersMutex.RUnlock()

	if info, ok := containers[cid]; ok {
		return info
	}

	for id, info := range containers {
		if strings.HasPrefix(id, cid) {
			docker.containersMutex.Lock()
			docker.containers[cid] = info
			docker.containersMutex.Unlock()
			return info
		}
	}

	return types.Container{}
}

func (docker DockerCli) List(ctx context.Context) []types.Container {
	cs, err := docker.cli.ContainerList(ctx, docker.listOptions)
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

	docker.containersMutex.Lock()
	defer docker.containersMutex.Unlock()
	for _, c := range containers {
		docker.containers[c.ID] = c
	}

	return containers
}

func (docker DockerCli) exist(ctx context.Context, cid, path string) bool {
	_, err := docker.cli.ContainerStatPath(ctx, cid, path)
	if err != nil {
		return false
	}
	return true
}

func (docker DockerCli) GetShell(ctx context.Context, cid string) string {
	for _, sh := range config.SHELL_LIST {
		if docker.exist(ctx, cid, sh) {
			logrus.Debugf("container [%s] us [%s]", cid, sh)
			return sh
		}
	}
	// generally it won't come so far
	return ""
}

func (docker DockerCli) Start(ctx context.Context, cid string) error {
	return docker.cli.ContainerStart(ctx, cid, apiTypes.ContainerStartOptions{})
}

func (docker DockerCli) Stop(ctx context.Context, cid string) error {
	// Notice: is there a need to config this stop duration?
	duration := time.Second * 5
	return docker.cli.ContainerStop(ctx, cid, &duration)
}

func (docker DockerCli) Restart(ctx context.Context, cid string) error {
	// restart immediately
	return docker.cli.ContainerRestart(ctx, cid, nil)
}

func buildListOptions(options string) (apiTypes.ContainerListOptions, error) {
	// ["-a", "-f", "key=val"]
	// https://docs.docker.com/engine/reference/commandline/ps/#filtering
	listOptions := apiTypes.ContainerListOptions{Filters: filters.NewArgs()}
	args := strings.Split(options, " ")
	for i, arg := range args {
		switch arg {
		case "-a", "--all":
			listOptions.All = true
		case "-f", "--filter":
			if i+1 < len(arg) {
				f := args[i+1]
				kv := strings.Split(f, "=")
				if len(kv) != 2 {
					return listOptions, fmt.Errorf("bad filter %s", f)
				}
				listOptions.Filters.Add(kv[0], kv[1])
			}
		case "-n", "--last":
			if i+1 < len(arg) {
				f := args[i+1]
				intF, err := strconv.Atoi(f)
				if err != nil {
					return listOptions, err
				}
				listOptions.Limit = intF
			}
		case "-l", "--latest":
			listOptions.Latest = true
		}
	}
	return listOptions, nil
}
