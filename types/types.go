package types

import "github.com/yudai/gotty/webtty"

type Container struct {
	// common
	ID, Name       string
	Image, Command string
	State, Status  string // "running"  "Up 13 minutes"
	IPs            []string
	Shell          string
	// k8s
	PodName, ContainerName string
	Namespace, RunningNode string
}

type ContainerActionMessage struct {
	Error   string `json:"err"`
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// Slave is webtty.Slave with some additional methods.
type TTY interface {
	webtty.Slave
	Exit() error
}

type InitMessage struct {
	Arguments string `json:"Arguments,omitempty"`
	AuthToken string `json:"AuthToken,omitempty"`
}
