package route

import (
	"io"

	"github.com/yudai/gotty/webtty"
)

type slaveWrapper struct {
	master io.ReadWriteCloser
}

func (sw *slaveWrapper) WindowTitleVariables() map[string]interface{} {
	return nil
}

func (sw *slaveWrapper) ResizeTerminal(columns int, rows int) error {
	return nil
}

func (sw *slaveWrapper) Write(p []byte) (n int, err error) {
	return sw.master.Write(p)
}

func (sw *slaveWrapper) Read(p []byte) (n int, err error) {
	return sw.master.Read(p)
}

func newSlave(master io.ReadWriteCloser) webtty.Slave {
	return &slaveWrapper{master: master}
}
