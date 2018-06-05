package types

type Container struct {
	// common
	ID, Name       string
	Image, Command string
	State, Status  string // "running"  "Up 13 minutes"
	IPs            []string
	// k8s
	PodName, ContainerName string
	Namespace, RunningNode string
}

type ContainerActionMessage struct {
	Error   string `json:"err"`
	Code    int    `json:"code"`
	Message string `json:"msg"`
}
