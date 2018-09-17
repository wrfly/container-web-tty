package route

import (
	"io"

	"github.com/yudai/gotty/webtty"
)

type slaveWrapper struct {
	rc io.ReadCloser
	pr io.ReadCloser
	pw io.WriteCloser
}

func (sw *slaveWrapper) WindowTitleVariables() map[string]interface{} {
	return nil
}

func (sw *slaveWrapper) ResizeTerminal(columns int, rows int) error {
	return nil
}

func (sw *slaveWrapper) Write(p []byte) (n int, err error) {
	if p[0] == 13 {
		p = append(p, 10) // append a new line
	}
	return sw.pw.Write(p)
}

func (sw *slaveWrapper) Read(p []byte) (n int, err error) {
	return sw.pr.Read(p)
}

func newSlave(rc io.ReadCloser) webtty.Slave {
	pr, pw := io.Pipe()
	go func() {
		defer pr.Close()
		defer pw.Close()
		defer rc.Close()

		bs := make([]byte, 2048)
		for {
			n, err := rc.Read(bs)
			if err != nil {
				return
			}
			pw.Write(bs[:n])
			if bs[n-1] == 10 {
				pw.Write([]byte{13})
			}
		}
	}()

	return &slaveWrapper{
		rc: rc,
		pr: pr,
		pw: pw,
	}
}
