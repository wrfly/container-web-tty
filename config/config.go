package config

type BackendConfig struct {
	Type        string   // docker or kubectl (for now)
	DockerPath  string   // default is /usr/bin/docker
	KubectlPath string   // default is /usr/bin/kubectl
	ExtraArgs   []string // extra args pass to docker or kubectl
}

type Config struct {
	Port    int
	Debug   bool
	Backend BackendConfig
	Servers []string // for proxy mode
}
