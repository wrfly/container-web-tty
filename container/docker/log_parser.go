package docker

import (
	"fmt"
	"io"
	"strconv"
)

type logReadCloser struct {
	io.Closer

	docker io.ReadCloser

	bsLeft int64

	prevHeader []byte
}

func _makeLine(bs []byte) []byte {
	if len(bs) == 0 {
		return nil
	}

	line := make([]byte, len(bs), len(bs)+1)
	copy(line, bs)

	if line[len(line)-1] == '\n' {
		line = append(line, '\r')
	}
	return line
}

// Read docker stream logs
// https://ahmet.im/blog/docker-logs-api-binary-format-explained/
func (rc *logReadCloser) Read(targetBytes []byte) (int, error) {
	n, err := rc.docker.Read(targetBytes)
	if err != nil {
		return n, err
	}

	// FIXME: quick and dirty solution
	// need to check the API version
	if n >= 4 {
		if targetBytes[1] != 0 || targetBytes[2] != 0 || targetBytes[3] != 0 {
			return n, nil
		}
	}

	p := make([]byte, len(targetBytes))
	copy(p, targetBytes)

	// docker log format
	var (
		msgLen int64
		start  int64
	)
	bs := make([]byte, 0, n)

	if len(rc.prevHeader) != 0 {
		p = append(rc.prevHeader, p...) // append previous header
		n = len(p)
		rc.prevHeader = nil
	} else if rc.bsLeft > 0 {
		line := _makeLine(p[:rc.bsLeft])
		bs = append(bs, line...)
		start = rc.bsLeft // reset line start index
		rc.bsLeft = 0     // reset left
	}

	for {
		if start+8 > int64(n) {
			rc.prevHeader = make([]byte, len(p[start:int64(n)]))
			copy(rc.prevHeader, p[start:int64(n)])
			break
		}

		header := p[start : start+8]
		if p[start] != 1 && p[start] != 2 {
			break
		}

		lenHex := fmt.Sprintf("%x", header[4:])
		msgLen, err = strconv.ParseInt(lenHex, 16, 64)
		if err != nil {
			return 0, err
		}

		start += 8 // move to msg beginning

		if start+msgLen > int64(n) {
			line := _makeLine(p[start:]) // append left bytes
			bs = append(bs, line...)
			rc.bsLeft = msgLen - (int64(n) - start)
			break
		}

		line := _makeLine(p[start : start+msgLen])
		bs = append(bs, line...)
		start += msgLen // move start position to next
	}

	return copy(targetBytes, bs), nil
}

func parseContainerLog(rc io.ReadCloser) io.ReadCloser {
	if rc == nil {
		return nil
	}
	return &logReadCloser{
		Closer: rc,
		docker: rc,
	}
}
