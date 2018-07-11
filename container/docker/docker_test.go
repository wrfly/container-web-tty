package docker

import (
	"context"
	"os"
	"testing"

	"github.com/wrfly/container-web-tty/config"
)

func TestDocker(t *testing.T) {
	if _, err := os.Open("/usr/bin/docker"); err != nil {
		t.Logf("docker cli not found, skip this test")
		return
	}

	ctx := context.Background()
	dockerConf := config.DockerConfig{
		DockerHost: "/var/run/docker.sock",
	}
	t.Run("test new docker client", func(t *testing.T) {
		cli, _, err := NewDockerCli(dockerConf)
		if err != nil {
			t.Error(err)
		}
		for _, c := range cli.List(ctx) {
			t.Logf("got container: %s", c.ID)
		}
	})
}
