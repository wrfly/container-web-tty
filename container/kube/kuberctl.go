package kube

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/types"
)

type KubeCli struct {
	cli        *kubernetes.Clientset
	config     *restclient.Config
	containers *types.Containers
}

func NewCli(conf config.KubeConfig, args []string) (*KubeCli, error) {
	// use the current context in kubeconfig
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", conf.ConfigPath)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	// get namespaces
	namespaceList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	ns := []string{}
	for _, namespace := range namespaceList.Items {
		ns = append(ns, namespace.Name)
	}
	logrus.Infof("New kube client: host [%s], namespaces [%s]",
		kubeConfig.Host, strings.Join(ns, ","))

	k := &KubeCli{
		cli:        clientset,
		containers: &types.Containers{},
		config:     kubeConfig,
	}
	k.List(context.Background())

	return k, nil
}

func (kube KubeCli) GetInfo(ctx context.Context, cid string) types.Container {
	if kube.containers.Len() == 0 {
		logrus.Debugf("zero containers, get cid %s", cid)
		kube.List(ctx)
	}

	// find in containers
	logrus.Debugf("find cid: %s", cid)
	container := kube.containers.Find(cid)
	if container.ID != "" {
		if container.Shell == "" {
			shell := kube.getShell(ctx, cid)
			kube.containers.SetShell(cid, shell)
			container.Shell = shell
		}
		return container
	}

	return types.Container{}
}

func trimContainerIDPrefix(id string) string {
	return strings.TrimLeft(id, "docker://")
}

func containerReady(ready bool) string {
	if ready {
		return "Ready"
	}
	return "Not Ready"
}

func containerStartTime(state v1.ContainerState) time.Duration {
	if state.Running == nil {
		return 0
	}
	return time.Since(state.Running.StartedAt.Time).Round(time.Second)
}

func (kube KubeCli) List(ctx context.Context) []types.Container {
	pods, err := kube.cli.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		logrus.Errorf("kubectl list pods error: %s", err)
		return nil
	}

	containers := []types.Container{}

	for _, pod := range pods.Items {
		// map key is name
		containerMap := make(map[string]types.Container, 0)

		spec := pod.Spec
		status := pod.Status

		podIP := pod.Status.PodIP
		hostIP := pod.Status.HostIP
		podState := string(status.Phase)

		// spec
		for _, container := range spec.Containers {
			c := types.Container{
				Command: strings.Join(container.Command, " "),
				Image:   container.Image,
			}
			containerMap[container.Name] = c
		}

		// status
		for _, container := range status.ContainerStatuses {
			id := trimContainerIDPrefix(container.ContainerID)
			if id == "" {
				continue
			}
			c := types.Container{
				ID:            id,
				PodName:       pod.GetName(),
				ContainerName: container.Name,
				Namespace:     pod.GetNamespace(),
				Name:          container.Name,
				State:         fmt.Sprintf("%s / %s", containerReady(container.Ready), podState),
				Status:        fmt.Sprintf("age: %s; restart %d", containerStartTime(container.State), container.RestartCount),
				IPs: func() []string {
					if podIP != hostIP {
						return []string{podIP, hostIP}
					}
					return []string{hostIP}
				}(),
				Image:   containerMap[container.Name].Image,
				Command: containerMap[container.Name].Command,
			}
			logrus.Debugf("get container: %+v\n", c)
			containers = append(containers, c)
		}
	}

	kube.containers.Set(containers)

	return containers
}

func (kube KubeCli) exist(ctx context.Context, containerID, path string) bool {
	info := kube.containers.Find(containerID)
	if info.ID == "" {
		return false
	}

	restClient := kube.cli.CoreV1().RESTClient()

	req := restClient.Post().
		Resource("pods").
		Name(info.PodName).
		Namespace(info.Namespace).
		SubResource("exec").
		Param("container", info.ContainerName).
		Param("command", "ls").
		Param("command", path).
		Param("stdout", "true").
		Param("stdin", "false").
		Param("tty", "false")

	logrus.Debugf("POST to %s", req.URL())
	exec, err := remotecommand.NewSPDYExecutor(kube.config, "POST", req.URL())
	if err != nil {
		logrus.Errorf("exist exec: setup executor error: [%v]", err)
		return false
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: new(bytes.Buffer),
		Tty:    false,
	})
	if err != nil {
		logrus.Debugf("exist exec error: [%v]", err)
		return false
	}

	logrus.Debugf("container %s exist %s", containerID, path)
	return true

}

func (kube KubeCli) getShell(ctx context.Context, cid string) string {
	logrus.Debugf("get container's shell path, cid: %s", cid)
	for _, sh := range config.SHELL_LIST {
		if kube.exist(ctx, cid, sh) {
			logrus.Debugf("get shell path %s", sh)
			return sh
		}
	}
	// generally it won't come so far
	return ""
}

func (kube KubeCli) Start(ctx context.Context, cid string) error {
	return nil
}

func (kube KubeCli) Stop(ctx context.Context, cid string) error {
	return nil
}

func (kube KubeCli) Restart(ctx context.Context, cid string) error {
	return nil
}

func (kube KubeCli) Exec(ctx context.Context, c types.Container) (types.TTY, error) {
	logrus.Debugf("exec pod: %v", c)
	if c.PodName == "" || c.Namespace == "" {
		return nil, fmt.Errorf("PodName or Namespace is empty")
	}
	pod, err := kube.cli.CoreV1().Pods(c.Namespace).
		Get(c.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if pod.Status.Phase == api.PodSucceeded || pod.Status.Phase == api.PodFailed {
		return nil,
			fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
	}

	restClient := kube.cli.CoreV1().RESTClient()

	req := restClient.Post().
		Resource("pods").
		Name(c.PodName).
		Namespace(c.Namespace).
		SubResource("exec").
		Param("container", c.ContainerName).
		Param("command", c.Shell).
		Param("stdin", "true").
		Param("stdout", "true").
		Param("tty", "true")

	enj := newInjector(ctx)

	logrus.Debugf("POST to %s", req.URL())
	exec, err := remotecommand.NewSPDYExecutor(kube.config, "POST", req.URL())
	if err != nil {
		return nil, err
	}

	go func() {
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:             enj.ttyIn,
			Stdout:            enj.ttyOut,
			Tty:               true,
			TerminalSizeQueue: enj.sq,
		})
		if err != nil {
			logrus.Errorf("exec error: [%v]", err)
		}
		logrus.Debug("exec done")
		// close in and out
		enj.ttyIn.Close()
		enj.ttyOut.Close()
	}()

	logrus.Debug("return enj")
	return &enj, nil
}
