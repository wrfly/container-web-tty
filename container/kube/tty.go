package kube

import (
	"context"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/remotecommand"
)

type execInjector struct {
	r      io.ReadCloser
	w      io.WriteCloser
	ttyIn  io.ReadCloser
	ttyOut io.WriteCloser

	resize resizeFunction
	sq     remotecommand.TerminalSizeQueue
}

func newInjector(ctx context.Context) execInjector {

	r, out := io.Pipe()
	in, w := io.Pipe()

	enj := execInjector{
		r:      r,
		w:      w,
		ttyIn:  in,
		ttyOut: out,
		sq: &sizeQueue{
			ctx:        ctx,
			resizeChan: make(chan remotecommand.TerminalSize),
		},
	}
	enj.resize = func(width int, height int) error {
		defer func() {
			recover()
		}()
		size := remotecommand.TerminalSize{
			Width:  uint16(width),
			Height: uint16(height),
		}
		enj.sq.(*sizeQueue).resizeChan <- size
		return nil
	}

	return enj
}

type resizeFunction func(width int, height int) error

func (enj *execInjector) Read(p []byte) (n int, err error) {
	return enj.r.Read(p)
}

func (enj *execInjector) Write(p []byte) (n int, err error) {
	return enj.w.Write(p)
}

func (enj *execInjector) Exit() error {
	// exit the shell
	enj.Write([]byte("exit\n"))

	// close the size queue
	enj.sq.(*sizeQueue).close()

	enj.r.Close()
	enj.w.Close()
	enj.ttyIn.Close()
	enj.ttyOut.Close()

	return nil
}

func (enj *execInjector) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{}
}

func (enj *execInjector) ResizeTerminal(width int, height int) error {
	logrus.Debugf("resize terminal to: %dx%d", width, height)
	var err error
	for i := 0; i < 3; i++ {
		if err = enj.resize(width, height); err == nil {
			break
		}
		time.Sleep(time.Millisecond * 5)
	}
	return err
}

type sizeQueue struct {
	ctx        context.Context
	resizeChan chan remotecommand.TerminalSize
}

func (s *sizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-s.resizeChan
	if !ok {
		return nil
	}
	return &size
}

func (s *sizeQueue) close() {
	close(s.resizeChan)
}
