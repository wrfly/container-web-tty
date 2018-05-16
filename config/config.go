package config

type DockerConfig struct {
	DockerPath string // default is /usr/bin/docker
	DockerHost string // default is /var/run/docker.sock
}

type KubeConfig struct {
	KubectlPath string // default is /usr/bin/kubectl
	KubeAPI     string // default is https://localhost:6443
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
	Kill    bool
}

type Config struct {
	Port     int
	LogLevel string
	Control  ControlConfig
	Backend  BackendConfig
	Servers  []string // for proxy mode
}
