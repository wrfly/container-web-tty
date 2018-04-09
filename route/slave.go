package route

import (
	gottyserver "github.com/yudai/gotty/server"
)

// Slave is webtty.Slave with some additional methods.
type Slave gottyserver.Slave

type Factory interface {
	Name() string
	New(args map[string][]string) (gottyserver.Slave, error)
}
