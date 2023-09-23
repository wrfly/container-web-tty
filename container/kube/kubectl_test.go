package kube

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/types"
)

func TestKubeCli(t *testing.T) {
	_, err := os.Stat("/home/mr/.kube/config")
	if err != nil {
		return
	}

	logrus.SetLevel(logrus.DebugLevel)

	k, err := NewCli(config.KubeConfig{ConfigPath: "/home/mr/.kube/config"})
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	execContainer := types.Container{}

	cs := k.List(ctx)
	for _, c := range cs {
		cc := k.GetInfo(ctx, c.ID)
		if cc.Shell != "" {
			execContainer = cc
			break
		}
	}

	tty, err := k.Exec(ctx, execContainer)
	if err != nil {
		t.Error(err)
		return
	}
	defer tty.Exit()

	time.Sleep(time.Second)
	_, err = tty.Write([]byte("pwd\n"))
	if err != nil {
		t.Error(err)
		return
	}

	p := make([]byte, 10)
	n, err := tty.Read(p)
	if err != nil {
		t.Logf("out: %s", p[:n])
		t.Error(err)
	}
}
