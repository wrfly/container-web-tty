package route

import (
	"io"

	"github.com/yudai/gotty/webtty"
)

type slaveWrapper struct {
	master io.ReadWriteCloser
}

func (sw *slaveWrapper) WindowTitleVariables() map[string]interface{} {
	return nil
}

func (sw *slaveWrapper) ResizeTerminal(columns int, rows int) error {
	return nil
}

func (sw *slaveWrapper) Write(p []byte) (n int, err error) {
	// logrus.Debugf("slaveWrapper write-> %s", p)
	// if p[0] == 13 {
	// 	p = append(p, 10) // append a new line
	// }
	return sw.master.Write(p)
}

func (sw *slaveWrapper) Read(p []byte) (n int, err error) {
	return sw.master.Read(p)
}

func newSlave(master io.ReadWriteCloser, share bool) webtty.Slave {
	// pr, pw := io.Pipe()
	// go func() {
	// 	defer pr.Close()
	// 	defer pw.Close()
	// 	defer master.Close()

	// 	bs := make([]byte, 2048)
	// 	for {
	// 		n, err := master.Read(bs)
	// 		if err != nil {
	// 			return
	// 		}
	// 		if share {
	// 			pw.Write(bs[:n])
	// 			continue
	// 		}
	// 		panic(1)
	// 		if n <= 1 {
	// 			continue
	// 		}

	// 		if n >= 2 && bs[n-2] == 13 { // \r\n
	// 			pw.Write(bs[:n])
	// 			continue
	// 		}

	// 		// only \n or a long log string contains \n
	// 		s, e := 0, 0
	// 		for e < n {
	// 			x := bytes.IndexByte(bs[e:n], 10)
	// 			if x == -1 {
	// 				break
	// 			}
	// 			s = e
	// 			e += x
	// 			pw.Write(bs[s:e])
	// 			pw.Write([]byte{13, 10})
	// 			e++ // skip this \n
	// 		}
	// 	}
	// }()

	return &slaveWrapper{
		master: master,
	}
}
