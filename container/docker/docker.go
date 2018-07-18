package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/types"
)

type DockerCli struct {
	cli         *client.Client
	containers  *types.Containers
	listOptions apiTypes.ContainerListOptions
}

func NewCli(conf config.DockerConfig, args []string) (*DockerCli, error) {
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
		return nil, err
	}

	listOptions, err := buildListOptions(conf.PsOptions)
	if err != nil {
		return nil, fmt.Errorf("build ps options error: %s", err)
	}
	logrus.Debugf("%+v", listOptions)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ping, err := cli.Ping(ctx)
	if err != nil {
		return nil, err
	}
	logrus.Infof("New docker client: OS [%s], API [%s]", ping.OSType, ping.APIVersion)
	dockerCli := &DockerCli{
		cli:         cli,
		containers:  &types.Containers{},
		listOptions: listOptions,
	}
	logrus.Infof("Warm up containers info...")
	dockerCli.List(ctx)

	return dockerCli, nil
}

func getContainerIP(networkSettings interface{}) (ips []string) {
	defer func() {
		if len(ips) == 0 {
			ips = []string{"null"}
		}
	}()

	if networkSettings == nil {
		return
	}
	switch networkSettings.(type) {
	case *apiTypes.SummaryNetworkSettings:
		network := networkSettings.(*apiTypes.SummaryNetworkSettings)
		for net := range network.Networks {
			if network.Networks[net] == nil {
				continue
			}
			ips = append(ips, network.Networks[net].IPAddress)
		}
	case *apiTypes.NetworkSettings:
		network := networkSettings.(*apiTypes.NetworkSettings)
		for net := range network.Networks {
			if network.Networks[net] == nil {
				continue
			}
			ips = append(ips, network.Networks[net].IPAddress)
		}
	}

	return
}

func (docker DockerCli) GetInfo(ctx context.Context, cid string) types.Container {
	if docker.containers.Len() == 0 {
		logrus.Debugf("zero containers, get cid %s", cid)
		docker.List(ctx)
	}

	// find in containers
	if container := docker.containers.Find(cid); container.ID != "" {
		if container.Shell == "" {
			shell := docker.getShell(ctx, cid)
			docker.containers.SetShell(cid, shell)
		}
		return container
	}

	// didn't get this container, this is rarelly happens
	cjson, err := docker.cli.ContainerInspect(ctx, cid)
	if err != nil {
		logrus.Errorf("inspect container %s error: %s", cid, err)
		return types.Container{}
	}

	c := docker.convertCjsonToContainre(cjson)
	if c.ID != "" {
		docker.containers.Append(c)
	}
	return c
}

func (docker DockerCli) convertCjsonToContainre(cjson apiTypes.ContainerJSON) types.Container {
	if cjson.Config == nil {
		// WTF?
		return types.Container{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	shell := docker.getShell(ctx, cjson.ID)

	c := types.Container{
		ID:      cjson.ID,
		Name:    cjson.Name,
		Image:   cjson.Image,
		Command: fmt.Sprintf("%s", cjson.Config.Cmd),
		IPs:     getContainerIP(cjson.NetworkSettings),
		Status:  cjson.State.Status,
		State:   cjson.State.Status,
		Shell:   shell,
	}

	return c
}

func (docker DockerCli) List(ctx context.Context) []types.Container {
	start := time.Now()
	logrus.Debug("list conatiners")
	cs, err := docker.cli.ContainerList(ctx, docker.listOptions)
	if err != nil {
		logrus.Errorf("list containers eror: %s", err)
		return nil
	}

	containers := make([]types.Container, len(cs))
	var (
		ips   []string
		shell string
	)
	for i, container := range cs {
		if old := docker.containers.Find(container.ID); old.ID != "" {
			shell = old.Shell
			ips = old.IPs
		} else {
			ips = getContainerIP(container.NetworkSettings)
		}
		containers[i] = types.Container{
			ID:      container.ID,
			Name:    container.Names[0][1:],
			Image:   container.Image,
			Command: container.Command,
			IPs:     ips,
			Status:  container.Status,
			State:   container.State,
			Shell:   shell,
		}
	}

	docker.containers.Set(containers)

	logrus.Debugf("list %d containers, use %s", len(containers), time.Now().Sub(start))
	return containers
}

func (docker DockerCli) exist(ctx context.Context, cid, path string) bool {
	_, err := docker.cli.ContainerStatPath(ctx, cid, path)
	if err != nil {
		return false
	}
	return true
}

func (docker DockerCli) getShell(ctx context.Context, cid string) string {
	for _, sh := range config.SHELL_LIST {
		if docker.exist(ctx, cid, sh) {
			logrus.Debugf("container [%s] use [%s]", cid, sh)
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

func (docker DockerCli) Exec(ctx context.Context, container types.Container) (types.TTY, error) {
	execConfig := apiTypes.ExecConfig{
		// User:         "string",
		Privileged:   false,
		Tty:          true,
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		// Env:          []string,
		Cmd: []string{container.Shell},
	}

	response, err := docker.cli.ContainerExecCreate(ctx, container.ID, execConfig)
	if err != nil {
		return nil, err
	}

	execID := response.ID
	if execID == "" {
		return nil, fmt.Errorf("exec ID empty")
	}

	resp, err := docker.cli.ContainerExecAttach(ctx, execID, execConfig)
	if err != nil {
		return nil, err
	}

	resizeFunc := func(width int, height int) error {
		err := docker.cli.ContainerExecResize(ctx, execID, apiTypes.ResizeOptions{
			Width:  uint(width),
			Height: uint(height),
		})
		if err != nil {
			logrus.Errorf("resize exec %s (container %s) window size to %dx%d; err: %v",
				container.ID, execID, width, height, err)
		}
		return err
	}

	return newExecInjector(resp, resizeFunc), nil
}
