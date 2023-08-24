package webtty

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"testing"
)

type pipePair struct {
	*io.PipeReader
	*io.PipeWriter
}

type slave struct {
	*io.PipeReader
	*io.PipeWriter
}

// WindowTitleVariables returns any values that can be used to fill out
// the title of a terminal.
func (s *slave) WindowTitleVariables() map[string]interface{} {
	return nil
}

// ResizeTerminal sets a new size of the terminal.
func (s *slave) ResizeTerminal(columns int, rows int) error {
	return nil
}

func TestWriteFromPTY(t *testing.T) {
	connMPipeReader, connMPipeWriter := io.Pipe()
	connSPipeReader, connSPipeWriter := io.Pipe()

	dt, err := New(pipePair{connMPipeReader, connMPipeWriter},
		&slave{connSPipeReader, connSPipeWriter})
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := dt.Run(ctx)
		if err == context.Canceled {
			return
		}
		if err != nil && err != context.Canceled {
			fmt.Printf("Unexpected error from Run(): %s\n", err)
		}
	}()

	message := []byte("foobar")

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		n, err := connMPipeReader.Read(buf)
		if err != nil {
			fmt.Printf("Unexpected error from Read(): %s\n", err)
			return
		}
		if buf[0] != Output {
			fmt.Printf("Unexpected message type `%c`\n", buf[0])
			return
		}

		decoded := make([]byte, 1024)
		n, err = base64.StdEncoding.Decode(decoded, buf[1:n])
		if err != nil {
			fmt.Printf("Unexpected error from Decode(): %s\n", err)
			return
		}
		if !bytes.Equal(decoded[:n], message) {
			fmt.Printf("Unexpected message received: `%s`\n", decoded[:n])
			return
		}
	}()

	n, err := dt.slave.Write(message)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if n != len(message) {
		t.Fatalf("Write() accepted `%d` for message `%s`", n, message)
	}

	cancel()
	wg.Wait()
}

func TestWriteFromConn(t *testing.T) {
	connInPipeReader, connInPipeWriter := io.Pipe()   // in to conn
	connOutPipeReader, connOutPipeWriter := io.Pipe() // out from conn

	_, connInPipeWriter2 := io.Pipe()  // in to conn
	connOutPipeReader2, _ := io.Pipe() // out from conn

	conn := pipePair{
		connOutPipeReader,
		connInPipeWriter,
	}
	dt, err := New(conn, &slave{
		connOutPipeReader2,
		connInPipeWriter2,
	})
	if err != nil {
		t.Fatalf("Unexpected error from New(): %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		err := dt.Run(ctx)
		if err != nil {
			fmt.Printf("Unexpected error from Run(): %s", err)
			return
		}
	}()

	var (
		message []byte
		n       int
	)
	readBuf := make([]byte, 1024)

	// input
	message = []byte("0hello\n") // line buffered canonical mode
	n, err = connOutPipeWriter.Write(message)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if n != len(message) {
		t.Fatalf("Write() accepted `%d` for message `%s`", n, message)
	}

	n, err = dt.slave.Read(readBuf)
	if err != nil {
		t.Fatalf("Unexpected error from Write(): %s", err)
	}
	if !bytes.Equal(readBuf[:n], message[1:]) {
		t.Fatalf("Unexpected message received: `%s`", readBuf[:n])
	}

	// ping
	message = []byte("1\n") // line buffered canonical mode
	n, err = connOutPipeWriter.Write(message)
	if n != len(message) {
		t.Fatalf("Write() accepted `%d` for message `%s`", n, message)
	}

	n, err = connInPipeReader.Read(readBuf)
	if err != nil {
		t.Fatalf("Unexpected error from Read(): %s", err)
	}
	if !bytes.Equal(readBuf[:n], []byte{'1'}) {
		t.Fatalf("Unexpected message received: `%s`", readBuf[:n])
	}

	// TODO: resize

	cancel()
	wg.Wait()
}
