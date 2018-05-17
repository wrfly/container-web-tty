package backend

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/sirupsen/logrus"
	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/types"
)

type KubeCli struct {
	cli             *kubernetes.Clientset
	containers      map[string]types.Container
	containersMutex *sync.RWMutex
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

	return &KubeCli{
		cli:             clientset,
		containers:      map[string]types.Container{},
		containersMutex: &sync.RWMutex{},
	}, []string{conf.KubectlPath, "exec", "-ti"}, nil

	// for {
	// 	pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	// 	if err != nil {
	// 		return nil, nil, err
	// 	}
	// 	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

	// 	// Examples for error handling:
	// 	// - Use helper functions like e.g. errors.IsNotFound()
	// 	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	// 	namespace := "default"
	// 	pod := "example-xxxxx"
	// 	_, err = clientset.CoreV1().Pods(namespace).Get(pod, metav1.GetOptions{})
	// 	if errors.IsNotFound(err) {
	// 		fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
	// 	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
	// 		fmt.Printf("Error getting pod %s in namespace %s: %v\n",
	// 			pod, namespace, statusError.ErrStatus.Message)
	// 	} else if err != nil {
	// 		return nil, nil, err
	// 	} else {
	// 		fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
	// 	}

	// 	time.Sleep(10 * time.Second)
	// }
}

// func getContainerIP(networkSettings *apiTypes.SummaryNetworkSettings) []string {
// 	ips := []string{}

// 	if networkSettings == nil {
// 		return ips
// 	}

// 	for net := range networkSettings.Networks {
// 		ips = append(ips, networkSettings.Networks[net].IPAddress)
// 	}

// 	return ips
// }

func (kube KubeCli) GetInfo(ID string) types.Container {
	if len(kube.containers) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		kube.List(ctx)
		cancel()
	}
	kube.containersMutex.RLock()
	defer kube.containersMutex.RUnlock()
	return kube.containers[ID]
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
				ID:        id,
				PodName:   pod.GetName(),
				Namespace: pod.GetNamespace(),
				Name:      container.Name,
				State:     fmt.Sprintf("%s / %s", containerReady(container.Ready), podState),
				Status:    fmt.Sprintf("age: %s; restart %d", containerStartTime(container.State), container.RestartCount),
				IPs:       []string{podIP, hostIP},
				Image:     containerMap[container.Name].Image,
				Command:   containerMap[container.Name].Command,
			}
			logrus.Debugf("get container: %+v\n", c)
			kube.containersMutex.Lock()
			kube.containersMutex.Unlock()
			kube.containers[id] = c
			containers = append(containers, c)
		}

	}

	return containers
}

func (kube KubeCli) exist(ctx context.Context, podname, path string) bool {
	// pod, err := kube.cli.CoreV1().Pods("").Get(podname, metav1.GetOptions{})
	// if err != nil {
	// 	logrus.Errorf("get pod [%s] error: %s", podname, err)
	// 	return false
	// }

	// if pod.Status.Phase == api.PodSucceeded || pod.Status.Phase == api.PodFailed {
	// 	return fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
	// }

	// 	containerName := p.ContainerName
	// 	if len(containerName) == 0 {
	// 		if len(pod.Spec.Containers) > 1 {
	// 			usageString := fmt.Sprintf("Defaulting container name to %s.", pod.Spec.Containers[0].Name)
	// 			if len(p.SuggestedCmdUsage) > 0 {
	// 				usageString = fmt.Sprintf("%s\n%s", usageString, p.SuggestedCmdUsage)
	// 			}
	// 			fmt.Fprintf(p.ErrOut, "%s\n", usageString)
	// 		}
	// 		containerName = pod.Spec.Containers[0].Name
	// 	}

	// 	// ensure we can recover the terminal while attached
	// t := p.setupTTY()

	// _, err := kube.cli.ContainerStatPath(ctx, cid, path)
	// if err != nil {
	// 	return false
	// }
	return true
}

func (kube KubeCli) GetShell(ctx context.Context, cid string) string {
	for _, sh := range types.SHELL_LIST {
		if kube.exist(ctx, cid, sh) {
			return sh
		}
	}
	// generally it would'n come here
	return "sh"
}
