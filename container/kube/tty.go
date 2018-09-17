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

	sq         *sizeQueue
	activeChan chan struct{}
}

func newInjector(ctx context.Context) execInjector {

	r, out := io.Pipe()
	in, w := io.Pipe()
	sq := &sizeQueue{
		ctx:        ctx,
		resizeChan: make(chan remotecommand.TerminalSize),
	}
	enj := execInjector{
		r:          r,
		w:          w,
		ttyIn:      in,
		ttyOut:     out,
		sq:         sq,
		activeChan: make(chan struct{}, 5),
	}

	return enj
}

type resizeFunction func(width int, height int) error

func (enj *execInjector) Read(p []byte) (n int, err error) {
	go func() {
		if len(enj.activeChan) == 0 {
			enj.activeChan <- struct{}{}
		}
	}()
	return enj.r.Read(p)
}

func (enj *execInjector) Write(p []byte) (n int, err error) {
	return enj.w.Write(p)
}

func (enj *execInjector) Exit() error {
	enj.Write([]byte("exit\n"))

	enj.r.Close()
	enj.w.Close()
	enj.ttyIn.Close()
	enj.ttyOut.Close()
	enj.sq.close()
	close(enj.activeChan)

	return nil
}

func (enj *execInjector) ActiveChan() <-chan struct{} {
	return enj.activeChan
}

func (enj *execInjector) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{}
}

func (enj *execInjector) ResizeTerminal(width int, height int) (err error) {
	logrus.Debugf("resize terminal to: %dx%d", width, height)
	for i := 0; i < 3; i++ {
		// there is a delay somehow, use this trick method to avoid it
		enj.sq.resize(width, height)
		time.Sleep(time.Millisecond * 50)
	}
	return
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

func (s *sizeQueue) resize(width int, height int) error {
	defer func() {
		recover()
	}()
	s.resizeChan <- remotecommand.TerminalSize{
		Width:  uint16(width),
		Height: uint16(height),
	}
	return nil
}
