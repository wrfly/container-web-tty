package remote

import (
	"io"
	"time"

	"github.com/sirupsen/logrus"
	pb "github.com/wrfly/container-web-tty/proxy/grpc"
)

// execInjector implement webtty.Slave
type execInjector struct {
	exec pb.ContainerServer_ExecClient

	r      io.ReadCloser
	w      io.WriteCloser
	ttyIn  io.ReadCloser
	ttyOut io.WriteCloser

	activeChan chan struct{}
}

type resizeFunction func(width int, height int) error

func newExecInjector(client pb.ContainerServer_ExecClient) *execInjector {
	r, out := io.Pipe()
	in, w := io.Pipe()
	return &execInjector{
		exec: client,

		r:      r,
		w:      w,
		ttyIn:  in,
		ttyOut: out,

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

	execOpts, err := enj.exec.Recv()
	if err != nil {
		return 0, err
	}
	logrus.Debugf("output: %s\n", execOpts.Cmd.Out)
	copy(p, execOpts.Cmd.Out)
	return len(execOpts.Cmd.Out), nil

}

func (enj *execInjector) Write(p []byte) (n int, err error) {
	logrus.Debugf("input: %s\n", p)
	return len(p), enj.exec.Send(&pb.ExecOptions{
		Cmd: &pb.Io{
			In: p,
		},
	})
}

func (enj *execInjector) Exit() error {
	enj.Write([]byte("exit\n"))
	enj.r.Close()
	enj.w.Close()
	enj.ttyIn.Close()
	enj.ttyOut.Close()

	close(enj.activeChan)
	return enj.exec.CloseSend()
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
func (enj *execInjector) resize(width int, height int) error {
	return enj.exec.Send(&pb.ExecOptions{
		Ws: &pb.WindowSize{
			Height: int32(height),
			Width:  int32(width),
		},
	})
}
