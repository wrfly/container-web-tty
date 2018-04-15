package types

type Container struct {
	ID      string
	Name    string
	Image   string
	Command string
	IPs     []string
	Status  string // "Up 13 minutes"
	State   string // "running"
}
