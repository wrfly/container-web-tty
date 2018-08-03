package remote

import (
	"time"

	pb "github.com/wrfly/container-web-tty/proxy/grpc"
)

// execWrapper implement webtty.Slave
type execWrapper struct {
	exec       pb.ContainerServer_ExecClient
	activeChan chan struct{}
}

type resizeFunction func(width int, height int) error

func newExecWrapper(client pb.ContainerServer_ExecClient) *execWrapper {
	return &execWrapper{
		exec:       client,
		activeChan: make(chan struct{}, 5),
	}
}

func (enj *execWrapper) Read(p []byte) (n int, err error) {
	go func() {
		if len(enj.activeChan) != 0 {
			return
		}
		enj.activeChan <- struct{}{}
	}()

	execOpts, err := enj.exec.Recv()
	if err != nil {
		return 0, err
	}
	// logrus.Debugf("output: %s\n", execOpts.Cmd.Out)
	copy(p, execOpts.Cmd.Out)
	return len(execOpts.Cmd.Out), nil

}

func (enj *execWrapper) Write(p []byte) (n int, err error) {
	// logrus.Debugf("input: %s\n", p)
	return len(p), enj.exec.Send(&pb.ExecOptions{
		Cmd: &pb.Io{
			In: p,
		},
	})
}

func (enj *execWrapper) Exit() error {
	enj.Write([]byte("exit\n"))
	close(enj.activeChan)
	return enj.exec.CloseSend()
}

func (enj *execWrapper) ActiveChan() <-chan struct{} {
	return enj.activeChan
}

func (enj *execWrapper) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{}
}

func (enj *execWrapper) ResizeTerminal(width int, height int) (err error) {
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
func (enj *execWrapper) resize(width int, height int) error {
	return enj.exec.Send(&pb.ExecOptions{
		Ws: &pb.WindowSize{
			Height: int32(height),
			Width:  int32(width),
		},
	})
}
