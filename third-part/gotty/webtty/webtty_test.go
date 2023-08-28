package webtty

import (
	"context"
	"encoding/base64"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type pipePair struct {
	*io.PipeReader
	*io.PipeWriter
}

type ptyPair struct {
	*io.PipeReader
	*io.PipeWriter
}

// WindowTitleVariables returns any values that can be used to fill out
// the title of a terminal.
func (s *ptyPair) WindowTitleVariables() map[string]interface{} { return nil }

// ResizeTerminal sets a new size of the terminal.
func (s *ptyPair) ResizeTerminal(int, int) error { return nil }

func TestWebTTY(t *testing.T) {
	socketReader, socketWriter := io.Pipe()
	ptyReader, ptyWriter := io.Pipe()

	socket := &ptyPair{socketReader, socketWriter}
	pty := &ptyPair{ptyReader, ptyWriter}

	ctx, cancel := context.WithCancel(context.Background())

	webTTY, err := New(socket, pty,
		WithPermitWrite(),
	)
	assert.Nil(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := webTTY.Run(ctx)
		if err == context.Canceled {
			return
		}
		assert.Nil(t, err)
	}()

	t.Run("write from socket", func(t *testing.T) {
		// 1 as Input
		message := []byte("1" + "foobar")

		start := make(chan bool)
		go func() {
			<-start
			n, err := socket.Write(message)
			assert.Nil(t, err)
			assert.Equal(t, n, len(message))
		}()

		close(start)
		buf := make([]byte, 1024)
		n, err := pty.Read(buf)
		assert.Nil(t, err)
		assert.EqualValues(t, string(message[1:]), string(buf[:n]))
	})

	t.Run("write from pty", func(t *testing.T) {
		message := []byte("hello")

		start := make(chan bool)
		go func() {
			<-start
			_, err := pty.Write(message)
			assert.Nil(t, err)
		}()

		close(start)
		buf := make([]byte, 1024)
		n, err := socket.Read(buf)
		assert.Nil(t, err)
		decoded, err := base64.StdEncoding.DecodeString(string(buf[1:n]))
		assert.Nil(t, err)
		assert.EqualValues(t, string(message), string(decoded))
	})

	cancel()
	wg.Wait()
}
