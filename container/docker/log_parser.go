package docker

import (
	"fmt"
	"io"
	"strconv"
)

type logReadCloser struct {
	io.Closer

	docker io.ReadCloser
}

// https://ahmet.im/blog/docker-logs-api-binary-format-explained/
func _dockerStreamFormat(p []byte) bool {
	if len(p) < 8 {
		return false
	}
	if p[0] != 1 && p[0] != 2 {
		return false
	}

	for _, x := range p[1:4] {
		if x != 0 {
			return false
		}
	}

	return true
}

func (rc logReadCloser) Read(p []byte) (int, error) {
	n, err := rc.docker.Read(p)
	if err != nil {
		return n, err
	}

	if !_dockerStreamFormat(p) {
		return n, err
	}

	// docker log format
	var (
		msgLen int64
		start  int64
	)
	bs := make([]byte, 0, len(p))
	for start+msgLen < int64(len(p)) {
		lenHex := fmt.Sprintf("%x", p[start+4:start+8])
		msgLen, err = strconv.ParseInt(lenHex, 16, 64)
		if err != nil {
			return 0, err
		}
		if msgLen == 0 {
			break
		}

		start += 8
		line := p[start : start+msgLen]
		if line[len(line)-1] == '\n' {
			line[len(line)-1] = '\r'
			line = append(line, '\n') // must be \r\n
		}
		bs = append(bs, line...)
		start += msgLen
	}

	copy(p, bs)
	return len(bs), nil
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
