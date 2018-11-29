package config

import "time"

var SHELL_LIST = [...]string{
	"/bin/bash",
	"/bin/ash",
	"/bin/sh",
}

type DockerConfig struct {
	DockerHost string // default is /var/run/docker.sock
	PsOptions  string
}

type KubeConfig struct {
	ConfigPath string // normally is $HOME/.kube/config
}

type GRPCConfig struct {
	Servers []string
	Auth    string
	Proxy   string // http or socks5
}

type BackendConfig struct {
	Type   string // docker or kubectl (for now)
	Docker DockerConfig
	Kube   KubeConfig
	GRPC   GRPCConfig
}

type ControlConfig struct {
	Enable  bool
	All     bool
	Start   bool
	Stop    bool
	Restart bool
}

type ServerConfig struct {
	Address  string
	Port     int
	GrpcPort int
	IdleTime time.Duration

	Credential      string
	EnableReconnect bool
	ReconnectTime   int
	MaxConnection   int
	WSOrigin        string
	Term            string `default:"xterm"`
	ShowLocation    bool
	EnableShare     bool

	// audit
	EnableAudit bool
	AuditLogDir string `default:"log"`

	Control ControlConfig

	// EnableBasicAuth bool `default:"false"`
	// Once            bool `default:"false"`
	// TitleVariables map[string]interface{}
}

type Config struct {
	Debug   bool
	Backend BackendConfig
	Server  ServerConfig
}

func New() *Config {
	return &Config{
		Backend: BackendConfig{
			Docker: DockerConfig{},
			Kube:   KubeConfig{},
			GRPC: GRPCConfig{
				Servers: []string{},
			},
		},
		Server: ServerConfig{
			Control: ControlConfig{},
		},
	}
}
