package types

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/yudai/gotty/webtty"
)

// TTY is webtty.Slave with some additional methods.
type TTY interface {
	webtty.Slave
	Exit() error
	// ActiveChan is to notify that the connection is active
	ActiveChan() <-chan struct{}
}

type ShareTTY struct {
	TTY
	shares map[string]shareTTY
	m      sync.Mutex
}

func (t *ShareTTY) Read(p []byte) (n int, err error) {
	n, e := t.TTY.Read(p)
	t.writeShares(p[:n])
	return n, e
}

func (t *ShareTTY) Write(p []byte) (n int, err error) {
	return t.TTY.Write(p)
}

func (t *ShareTTY) Close() error {
	t.m.Lock()
	for _, s := range t.shares {
		s.Close()
	}
	t.m.Unlock()

	return nil
}

func (t *ShareTTY) Exit() error {
	if err := t.Close(); err != nil {
		return err
	}
	return t.TTY.Exit()
}

func (t *ShareTTY) Fork(clientIP string) io.ReadCloser {
	ID := fmt.Sprintf("%s-%d", clientIP, time.Now().UnixNano())
	s := newShareTTY(t, ID)
	t.m.Lock()
	t.shares[ID] = s
	t.m.Unlock()
	return &s
}

func (t *ShareTTY) writeShares(bs []byte) {
	t.m.Lock()
	for _, s := range t.shares {
		s.pw.Write(bs)
	}
	t.m.Unlock()
}

func NewShareTTY(t TTY) *ShareTTY {
	return &ShareTTY{
		TTY:    t,
		shares: make(map[string]shareTTY, 50),
	}
}

type shareTTY struct {
	pt *ShareTTY
	id string
	pr *io.PipeReader
	pw *io.PipeWriter
}

func (s *shareTTY) Close() error {
	// delete itself
	parent := s.pt
	parent.m.Lock()
	delete(parent.shares, s.id)
	parent.m.Unlock()

	// close pr & pw
	s.pr.Close()
	return s.pw.Close()
}

func (s *shareTTY) Read(p []byte) (int, error) {
	return s.pr.Read(p)
}

func newShareTTY(parent *ShareTTY, id string) shareTTY {
	s := shareTTY{
		pt: parent,
		id: id,
	}
	s.pr, s.pw = io.Pipe()
	return s
}
