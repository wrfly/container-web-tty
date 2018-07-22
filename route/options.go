package route

import (
	"time"

	"github.com/wrfly/container-web-tty/config"
)

type Options struct {
	Address         string `default:"0.0.0.0"`
	Port            string `default:"8080"`
	Credential      string `default:""`
	EnableReconnect bool   `default:"false"`
	ReconnectTime   int    `default:"10"`
	MaxConnection   int    `default:"0"`
	WSOrigin        string `default:""`
	Term            string `default:"xterm"`
	Timeout         time.Duration
	Control         config.ControlConfig

	// EnableBasicAuth bool `default:"false"`
	// Once            bool `default:"false"`
	// TitleVariables map[string]interface{}
}
