package docker

import (
	"time"

	apiTypes "github.com/docker/docker/api/types"
)

// execInjector implement webtty.Slave
type execInjector struct {
	hResp      apiTypes.HijackedResponse
	resize     resizeFunction
	activeChan chan struct{}
}

type resizeFunction func(width int, height int) error

func newExecInjector(resp apiTypes.HijackedResponse, resize resizeFunction) *execInjector {
	return &execInjector{
		hResp:      resp,
		resize:     resize,
		activeChan: make(chan struct{}, 5),
	}
}

func (enj *execInjector) Read(p []byte) (n int, err error) {
	go func() {
		if len(enj.activeChan) != 0 {
			return
		}
		enj.activeChan <- struct{}{}
	}()
	// logrus.Debugf("output: %s\n", p)
	return enj.hResp.Reader.Read(p)
}

func (enj *execInjector) Write(p []byte) (n int, err error) {
	// logrus.Debugf("input: %v\n", p)
	return enj.hResp.Conn.Write(p)
}

func (enj *execInjector) Exit() error {
	enj.Write([]byte{3}) // ^C
	enj.Write([]byte{4}) // ^D
	close(enj.activeChan)
	return enj.hResp.Conn.Close()
}

func (enj *execInjector) ActiveChan() <-chan struct{} {
	return enj.activeChan
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
