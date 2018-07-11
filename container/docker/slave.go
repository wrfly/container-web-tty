package docker

import (
	"time"

	apiTypes "github.com/docker/docker/api/types"
)

// // Slave represents a PTY slave, typically it's a local command.
// type Slave interface {
// 	io.ReadWriter

// 	// WindowTitleVariables returns any values that can be used to fill out
// 	// the title of a terminal.
// 	WindowTitleVariables() map[string]interface{}

// 	// ResizeTerminal sets a new size of the terminal.
// 	ResizeTerminal(columns int, rows int) error
// }

// execInjector implement webtty.Slave
type execInjector struct {
	hResp  apiTypes.HijackedResponse
	resize resizeFunction
}

type resizeFunction func(width int, height int) error

func newExecInjector(resp apiTypes.HijackedResponse, resize resizeFunction) *execInjector {
	return &execInjector{
		hResp:  resp,
		resize: resize,
	}
}

func (enj *execInjector) Read(p []byte) (n int, err error) {
	return enj.hResp.Reader.Read(p)
}

func (enj *execInjector) Write(p []byte) (n int, err error) {
	return enj.hResp.Conn.Write(p)
}

func (enj *execInjector) Close() error {
	return enj.hResp.Conn.Close()
}

func (enj *execInjector) Exit() error {
	enj.Write([]byte("exit\n"))
	return enj.hResp.Conn.Close()
}

func (enj *execInjector) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{}
}

func (enj *execInjector) ResizeTerminal(width int, height int) error {
	// since the process may not up so fast, give it 15ms
	// retry 3 times
	var err error
	for i := 0; i < 3; i++ {
		if err = enj.resize(width, height); err == nil {
			break
		}
		time.Sleep(time.Millisecond * 5)
	}
	return err
}
