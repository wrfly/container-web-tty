package types

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/wrfly/pubsub"
	"github.com/yudai/gotty/webtty"
)

var globalPubSuber = pubsub.NewMemPubSuber()

// TTY is webtty.Slave with some additional methods.
type TTY interface {
	webtty.Slave
	Exit() error
	// ActiveChan is to notify that the connection is active
	ActiveChan() <-chan struct{}
}

type SlaveTTY struct {
	ps  pubsub.PubSubChan
	tty TTY

	readOnly bool

	masterOutputs []byte
}

func (s *SlaveTTY) Read(p []byte) (int, error) {
	if len(s.masterOutputs) != 0 {
		copy(p[:len(s.masterOutputs)], s.masterOutputs)
		s.masterOutputs = nil
		return len(p), nil
	}

	bs := <-s.ps.Read()
	// logrus.Debugf("slave tty read: %s", bs)
	copy(p[:len(bs)], bs)
	return len(bs), nil
}

func (s *SlaveTTY) Write(p []byte) (int, error) {
	// logrus.Debugf("slave tty write: %s", p)
	if !s.readOnly {
		s.tty.Write(p)
	}
	return len(p), nil // write to parent as well
}

func (s *SlaveTTY) Close() error {
	logrus.Debugf("close slave tty")
	return nil
}

type MasterTTY struct {
	TTY
	id      string
	pubC    pubsub.PubChan
	outputs []byte // previous outputs
}

func (m *MasterTTY) Read(p []byte) (n int, err error) {
	n, err = m.TTY.Read(p) // read from tty
	// logrus.Debugf("read from container: %s", p[:n])

	// publish to all, ignore the error
	m.pubC.Write(p[:n])

	return
}

func (m *MasterTTY) Write(p []byte) (n int, err error) {
	// logrus.Debugf("read from master: %x", p)
	m.TTY.Write(p) // write to container
	return len(p), nil
}

func (m *MasterTTY) Close() error {
	logrus.Debugf("close master/fork-master tty: %s", m.id)
	return nil
}

func (m *MasterTTY) Fork(ctx context.Context, collaborate bool) *SlaveTTY {
	pubsub, err := globalPubSuber.PubSub(ctx, m.id)
	if err != nil {
		panic(err) // shouldn't happen
	}
	outputs := make([]byte, len(m.outputs))
	copy(outputs, m.outputs)
	return &SlaveTTY{
		tty: m.TTY,
		ps:  pubsub,
		// options
		readOnly: !collaborate,
		// previous outputs from master
		masterOutputs: outputs,
	}
}

func NewMasterTTY(ctx context.Context, t TTY, execID string) (*MasterTTY, error) {
	pubsub, err := globalPubSuber.PubSub(ctx, execID)
	if err != nil {
		return nil, err
	}

	master := &MasterTTY{
		TTY:     t,
		id:      execID,
		pubC:    pubsub,
		outputs: make([]byte, 1e3),
	}

	go func() {
		for output := range pubsub.Read() {
			master.outputs = append(master.outputs, output...)
			// master.outputs = append(master.outputs, '\n')
			if len(master.outputs) > 1e3 {
				master.outputs = master.outputs[len(master.outputs)-1e3:]
			}
		}
	}()

	return master, nil
}
