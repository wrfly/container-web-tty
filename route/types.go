package route

import (
	"context"

	gottyserver "github.com/yudai/gotty/server"
)

type InitMessage struct {
	Arguments string `json:"Arguments,omitempty"`
	AuthToken string `json:"AuthToken,omitempty"`
}

// Slave is webtty.Slave with some additional methods.
type Slave gottyserver.Slave

type Factory interface {
	Name() string
	New(args map[string][]string) (gottyserver.Slave, error)
}

// RunOptions holds a set of configurations for Server.Run().
type RunOptions struct {
	gracefulCtx context.Context
}

// RunOption is an option of Server.Run().
type RunOption func(*RunOptions)

// WithGracefullContext accepts a context to shutdown a Server
// with care for existing client connections.
func WithGracefullContext(ctx context.Context) RunOption {
	return func(options *RunOptions) {
		options.gracefulCtx = ctx
	}
}
