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
}

func (s *SlaveTTY) Read(p []byte) (int, error) {
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
	id   string
	pubC pubsub.PubChan
}

func (m *MasterTTY) Read(p []byte) (n int, err error) {
	n, err = m.TTY.Read(p) // read from tty
	// logrus.Debugf("read from container: %s", p[:n])

	// publish to all
	if err := m.pubC.Write(p[:n]); err != nil {
		panic(err)
	}
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
	return &SlaveTTY{
		tty: m.TTY,
		ps:  pubsub,
		// options
		readOnly: !collaborate,
	}
}

func NewMasterTTY(ctx context.Context, t TTY, shareID string) (*MasterTTY, error) {
	pubChan, err := globalPubSuber.Pub(ctx, shareID)
	if err != nil {
		return nil, err
	}

	return &MasterTTY{
		TTY:  t,
		id:   shareID,
		pubC: pubChan,
	}, nil
}
