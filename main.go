package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/config"
)

func envVars(e string) []string {
	e = strings.ToUpper(e)
	return []string{"WEB_TTY_" + strings.Replace(e, "-", "_", -1)}
}

func main() {
	conf := config.Config{
		Backend: config.BackendConfig{
			Docker: config.DockerConfig{},
			Kube:   config.KuberConfig{},
		},
	}
	appFlags := []cli.Flag{
		&cli.IntFlag{
			Name:        "port",
			Aliases:     []string{"p"},
			EnvVars:     envVars("port"),
			Usage:       "server port",
			Value:       8080,
			Destination: &conf.Port,
		},
		&cli.StringFlag{
			Name:        "log-level",
			Aliases:     []string{"l"},
			Value:       "info",
			EnvVars:     envVars("log-level"),
			Usage:       "log level",
			Destination: &conf.LogLevel,
		},
		&cli.StringFlag{
			Name:        "backend",
			Aliases:     []string{"b"},
			EnvVars:     envVars("backend"),
			Value:       "docker",
			Usage:       "backend type, docker or kubectl for now",
			Destination: &conf.Backend.Type,
		},
		&cli.StringFlag{
			Name:        "docker-path",
			EnvVars:     envVars("docker-path"),
			Value:       "/usr/bin/docker",
			Usage:       "docker cli path",
			Destination: &conf.Backend.Docker.DockerPath,
		},
		&cli.StringFlag{
			Name:        "docker-sock",
			EnvVars:     envVars("docker-sock"),
			Value:       "/var/run/docker.sock",
			Usage:       "docker sock path",
			Destination: &conf.Backend.Docker.DockerSock,
		},
		&cli.StringFlag{
			Name:        "kubectl-path",
			EnvVars:     envVars("kubectl-path"),
			Value:       "/usr/bin/kubectl",
			Usage:       "kubectl cli path",
			Destination: &conf.Backend.Kube.KubectlPath,
		},
		&cli.StringSliceFlag{
			Name:    "extra-args",
			EnvVars: envVars("extra-args"),
			Usage:   "extra args for your backend",
		},
		&cli.StringSliceFlag{
			Name:    "servers",
			EnvVars: envVars("servers"),
			Usage:   "upstream servers, for proxy mode",
		},
		&cli.BoolFlag{
			Name:    "help",
			Aliases: []string{"h"},
			Usage:   "show help",
		},
	}

	app := &cli.App{
		Name:      "container-web-tty",
		Usage:     "connect your containers via a web-tty",
		UsageText: "container-web-tty [global options]",
		Flags:     appFlags,
		HideHelp:  true,
		Authors:   author,
		Version:   fmt.Sprintf("Version: %s\tCommit: %s\tDate: %s", Version, CommitID, BuildAt),
		Action: func(c *cli.Context) error {
			if c.Bool("help") {
				return cli.ShowAppHelp(c)
			}

			conf.Backend.ExtraArgs = c.StringSlice("extra-args")
			conf.Servers = c.StringSlice("servers")
			level, err := logrus.ParseLevel(conf.LogLevel)
			if err != nil {
				logrus.Error(err)
				return err
			}
			logrus.SetLevel(level)
			logrus.Debugf("got config: %+v", conf)

			run(c, conf)
			return nil
		},
	}

	app.Run(os.Args)
}
