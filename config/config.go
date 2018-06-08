package config

var SHELL_LIST = []string{"/bin/bash", "/bin/ash", "/bin/sh"}

type DockerConfig struct {
	DockerPath string // default is /usr/bin/docker
	DockerHost string // default is /var/run/docker.sock
	PsOptions  string
}

type KubeConfig struct {
	KubectlPath string // default is /usr/bin/kubectl
	ConfigPath  string // normally is $HOME/.kube/config
}

type BackendConfig struct {
	Type      string // docker or kubectl (for now)
	Docker    DockerConfig
	Kube      KubeConfig
	ExtraArgs []string // extra args pass to docker or kubectl
}

type ControlConfig struct {
	Enable  bool
	Start   bool
	Stop    bool
	Restart bool
}

type Config struct {
	Port     int
	LogLevel string
	Control  ControlConfig
	Backend  BackendConfig
	Servers  []string // for proxy mode
}
