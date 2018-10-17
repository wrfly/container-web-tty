package route

import (
	"time"

	"github.com/wrfly/container-web-tty/config"
)

type Options struct {
	Address         string
	Port            string
	Credential      string
	EnableReconnect bool
	ReconnectTime   int
	MaxConnection   int
	WSOrigin        string
	Term            string `default:"xterm"`
	Timeout         time.Duration
	Control         config.ControlConfig
	ShowLocation    bool
	EnableShare     bool

	// EnableBasicAuth bool `default:"false"`
	// Once            bool `default:"false"`
	// TitleVariables map[string]interface{}
}
