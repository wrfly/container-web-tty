package backend

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/types"
)

type KubeCli struct {
	cli                 *kubernetes.Clientset
	containers          map[string]types.Container
	containersMutex     *sync.RWMutex
	binPath, configPath string
}

func NewKubeCli(conf config.KubeConfig) (*KubeCli, []string, error) {
	// use the current context in kubeconfig
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", conf.ConfigPath)
	if err != nil {
		return nil, nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, nil, err
	}

	// get namespaces
	namespaceList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, err
	}
	ns := []string{}
	for _, namespace := range namespaceList.Items {
		ns = append(ns, namespace.Name)
	}
	logrus.Infof("New kube client: host [%s], namespaces [%s]",
		kubeConfig.Host, strings.Join(ns, ","))

	oriConfigPath, exist := os.LookupEnv("KUBECONFIG")
	if !exist {
		logrus.Debugf("original kube config path not exist, set to %s", conf.ConfigPath)
		os.Setenv("KUBECONFIG", conf.ConfigPath)
	} else if oriConfigPath != conf.ConfigPath {
		logrus.Debugf("original kube config path %s != %s", oriConfigPath, conf.ConfigPath)
		os.Setenv("KUBECONFIG", "$KUBECONFIG:"+conf.ConfigPath)
	}
	logrus.Infof("kube config path: %s", os.Getenv("KUBECONFIG"))

	return &KubeCli{
		cli:             clientset,
		containers:      map[string]types.Container{},
		containersMutex: &sync.RWMutex{},
		binPath:         conf.KubectlPath,
		configPath:      conf.ConfigPath,
	}, []string{conf.KubectlPath, "exec", "-ti"}, nil
}

func (kube KubeCli) GetInfo(ctx context.Context, cid string) types.Container {
	if len(kube.containers) == 0 {
		kube.List(ctx)
	}
	kube.containersMutex.RLock()
	containers := kube.containers
	kube.containersMutex.RUnlock()

	if info, ok := containers[cid]; ok {
		return info
	}

	for id, info := range containers {
		if strings.HasPrefix(id, cid) {
			kube.containersMutex.Lock()
			kube.containers[cid] = info
			kube.containersMutex.Unlock()
			return info
		}
	}

	return types.Container{}
}

func trimContainerIDPrefix(id string) string {
	if id := strings.TrimLeft(id, "docker://"); id != "" {
		return id
	}
	return "null"
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
			kube.containersMutex.Lock()
			kube.containers[id] = c
			kube.containersMutex.Unlock()
			containers = append(containers, c)
		}

	}

	return containers
}

func (kube KubeCli) exist(ctx context.Context, containerID, path string) bool {
	info := kube.GetInfo(ctx, containerID)
	if info.ID != containerID {
		logrus.Errorf("get container [%s]'s info error, real ID: %s", containerID, info.ID)
		return false
	}
	logrus.Debugf("container info: %v", info)

	pn := info.PodName
	ns := info.Namespace
	cn := info.ContainerName

	args := []string{"-n", ns, "exec", pn, "-c", cn, "ls", path}
	cmd := exec.Command(kube.binPath, args...)
	if err := cmd.Run(); err != nil {
		logrus.Errorf("run cmd %s %s error: %s", kube.binPath, args, err)
		return false
	}
	return true
}

func (kube KubeCli) GetShell(ctx context.Context, cid string) string {
	logrus.Debugf("get container's shell path, cid: %s", cid)
	for _, sh := range config.SHELL_LIST {
		if kube.exist(ctx, cid, sh) {
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
