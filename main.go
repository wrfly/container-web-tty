package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/config"
)

func main() {
	conf := config.Config{
		Backend: config.BackendConfig{
			Docker: config.DockerConfig{},
			Kube:   config.KubeConfig{},
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
			Usage:       "backend type, 'docker' or 'kube' for now",
			Destination: &conf.Backend.Type,
		},
		&cli.StringFlag{
			Name:        "docker-path",
			EnvVars:     envVars("docker-path"),
			Value:       dockerCliPath(),
			Usage:       "docker cli path",
			Destination: &conf.Backend.Docker.DockerPath,
		},
		&cli.StringFlag{
			Name:        "docker-host",
			EnvVars:     envVars("docker-host"),
			Value:       "/var/run/docker.sock",
			Usage:       "docker host path",
			Destination: &conf.Backend.Docker.DockerHost,
		},
		&cli.StringFlag{
			Name:        "kubectl-path",
			EnvVars:     envVars("kubectl-path"),
			Value:       "/usr/bin/kubectl",
			Usage:       "kubectl cli path",
			Destination: &conf.Backend.Kube.KubectlPath,
		},
		&cli.StringFlag{
			Name:        "kube-api",
			EnvVars:     envVars("kube-api"),
			Value:       "https://localhost:6443",
			Usage:       "kubectl api address",
			Destination: &conf.Backend.Kube.KubeAPI,
		},
		&cli.StringFlag{
			Name:        "kube-config",
			EnvVars:     envVars("kube-config"),
			Value:       kubeConfigPath(),
			Usage:       "kube config path",
			Destination: &conf.Backend.Kube.ConfigPath,
		},
		&cli.StringFlag{
			Name:    "extra-args",
			EnvVars: envVars("extra-args"),
			Usage:   "pass extra args to the backend",
		},
		&cli.StringFlag{
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
		Version: fmt.Sprintf("version: %s\tcommit: %s\tdate: %s",
			Version, CommitID, BuildAt),
		Action: func(c *cli.Context) error {
			if c.Bool("help") {
				return cli.ShowAppHelp(c)
			}

			if eArgs := c.String("extra-args"); eArgs != "" {
				conf.Backend.ExtraArgs = strings.Split(eArgs, " ")
			} else {
				switch conf.Backend.Type {
				case "docker":
					defaultArgs := "-e HISTCONTROL=ignoredups -e TERM=xterm"
					conf.Backend.ExtraArgs = strings.Split(defaultArgs, " ")
				case "kube":
				}
			}

			conf.Servers = strings.Split(c.String("servers"), " ")
			level, err := logrus.ParseLevel(conf.LogLevel)
			if err != nil {
				logrus.Error(err)
				return err
			}
			logrus.SetLevel(level)
			if level != logrus.DebugLevel {
				gin.SetMode(gin.ReleaseMode)
			}
			logrus.Debugf("got config: %+v", conf)

			run(c, conf)
			return nil
		},
	}

	app.Run(os.Args)
}
