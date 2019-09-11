package route

import (
	"bytes"
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

func newSlave(rc io.ReadCloser, share bool) webtty.Slave {
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
			if share {
				pw.Write(bs[:n])
				continue
			}

			if n <= 1 {
				continue
			}

			if n >= 2 && bs[n-2] == 13 { // \r\n
				pw.Write(bs[:n])
				continue
			}

			// only \n or a long log string contains \n
			s, e := 0, 0
			for e < n {
				x := bytes.IndexByte(bs[e:n], 10)
				if x == -1 {
					break
				}
				s = e
				e += x
				pw.Write(bs[s:e])
				pw.Write([]byte{13, 10})
				e++ // skip this \n
			}
		}
	}()

	return &slaveWrapper{
		rc: rc,
		pr: pr,
		pw: pw,
	}
}
