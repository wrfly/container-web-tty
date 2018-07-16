package kube

import (
	"context"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/remotecommand"
)

type execInjector struct {
	r      io.Reader
	w      io.Writer
	resize resizeFunction
	sq     remotecommand.TerminalSizeQueue
}

func newInjector(ctx context.Context, r io.Reader, w io.Writer) execInjector {
	enj := execInjector{
		r: r,
		w: w,
		sq: &sizeQueue{
			ctx:        ctx,
			resizeChan: make(chan remotecommand.TerminalSize),
		},
	}
	enj.resize = func(width int, height int) error {
		defer func() {
			x := recover()
			logrus.Errorf("got a panic in resize func: %v", x)
		}()

		enj.sq.(*sizeQueue).resizeChan <- remotecommand.TerminalSize{
			Width:  uint16(width),
			Height: uint16(height),
		}

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
	enj.Write([]byte("exit\n"))
	close(enj.sq.(*sizeQueue).resizeChan)
	return nil
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
