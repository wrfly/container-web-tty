package config

import "time"

var SHELL_LIST = []string{"/bin/bash", "/bin/ash", "/bin/sh"}

type DockerConfig struct {
	DockerHost string // default is /var/run/docker.sock
	PsOptions  string
}

type KubeConfig struct {
	ConfigPath string // normally is $HOME/.kube/config
}

type RemoteConfig struct {
	Servers []string
	Auth    string
}

type BackendConfig struct {
	Type      string // docker or kubectl (for now)
	Docker    DockerConfig
	Kube      KubeConfig
	Remote    RemoteConfig
	ExtraArgs []string // extra args pass to docker or kubectl
}

type ControlConfig struct {
	Enable  bool
	All     bool
	Start   bool
	Stop    bool
	Restart bool
}

type ServerConfig struct {
	Addr     string
	Port     int
	GrpcPort int
	IdleTime time.Duration
}

type Config struct {
	Debug   bool
	Control ControlConfig
	Backend BackendConfig
	Server  ServerConfig
}

func New() *Config {
	return &Config{
		Backend: BackendConfig{
			Docker: DockerConfig{},
			Kube:   KubeConfig{},
			Remote: RemoteConfig{
				Servers: []string{},
			},
		},
		Server:  ServerConfig{},
		Control: ControlConfig{},
	}
}
