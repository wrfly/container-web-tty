package docker

import (
	"time"

	apiTypes "github.com/docker/docker/api/types"
)

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

func (enj *execInjector) ResizeTerminal(width int, height int) (err error) {
	// since the process may not up so fast, give it 150ms
	// retry 3 times
	for i := 0; i < 3; i++ {
		if err = enj.resize(width, height); err == nil {
			return
		}
		time.Sleep(time.Millisecond * 50)
	}
	return
}
