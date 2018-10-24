package audit

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type LogOpts struct {
	Dir, ContainerID, ClientIP string
}

func LogTo(ctx context.Context, r io.Reader, opts LogOpts) {
	logDir := opts.Dir
	if !strings.HasPrefix(logDir, "/") {
		pwd, err := os.Getwd()
		if err != nil {
			logrus.Errorf("audit get pwd error: %s", err)
			return
		}
		logDir = path.Join(pwd, logDir)
	}

	logDir = path.Join(logDir, opts.ContainerID[:12])
	_, err := os.Stat(logDir)
	if os.IsNotExist(err) {
		logrus.Debugf("create dir %s", logDir)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			logrus.Errorf("mkdir error: %s", err)
			return
		}
	}
	fPath := path.Join(logDir, fmt.Sprintf("%s-%d.log",
		strings.Split(opts.ClientIP, ":")[0], time.Now().Unix()),
	)

	f, err := os.Create(fPath)
	if err != nil {
		logrus.Errorf("audit create file [%s] error: %s", fPath, err)
		return
	}
	defer f.Close()

	buff := make([]byte, 2048)
	var start int64
	for ctx.Err() == nil {
		n, err := r.Read(buff)
		if err != nil {
			if err == io.EOF {
				return
			}
			logrus.Errorf("audit read container error: %s", err)
			return
		}

		_, err = f.WriteAt(buff[:n], start)
		if err != nil {
			logrus.Errorf("audit write file error: %s", err)
			return
		}
		start += int64(n)
	}
}
