package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/util"
)

func main() {
	conf := config.New()
	appFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "addr",
			EnvVars:     util.EnvVars("address"),
			Usage:       "server binding address",
			Value:       "0.0.0.0",
			Destination: &conf.Server.Address,
		},
		&cli.IntFlag{
			Name:        "port",
			Aliases:     []string{"p"},
			EnvVars:     util.EnvVars("port"),
			Usage:       "HTTP server port, -1 for disable the HTTP server",
			Value:       8080,
			Destination: &conf.Server.Port,
		},
		&cli.BoolFlag{
			Name:        "debug",
			Aliases:     []string{"d"},
			Value:       false,
			EnvVars:     util.EnvVars("debug"),
			Usage:       "debug mode (log-level=debug enable pprof)",
			Destination: &conf.Debug,
		},
		&cli.StringFlag{
			Name:        "backend",
			Aliases:     []string{"b"},
			EnvVars:     util.EnvVars("backend"),
			Value:       "docker",
			Usage:       "backend type, 'docker' or 'kube' or 'grpc'(remote)",
			Destination: &conf.Backend.Type,
		},
		&cli.StringFlag{
			Name:        "docker-host",
			EnvVars:     append(util.EnvVars("docker-host"), "DOCKER_HOST"),
			Value:       "/var/run/docker.sock",
			Usage:       "docker host path",
			Destination: &conf.Backend.Docker.DockerHost,
		},
		&cli.StringFlag{
			Name:        "docker-ps",
			EnvVars:     util.EnvVars("docker-ps"),
			Usage:       "docker ps options",
			Destination: &conf.Backend.Docker.PsOptions,
		},
		&cli.StringFlag{
			Name:        "kube-config",
			EnvVars:     util.EnvVars("kube-config"),
			Value:       util.KubeConfigPath(),
			Usage:       "kube config path",
			Destination: &conf.Backend.Kube.ConfigPath,
		},
		&cli.IntFlag{
			Name:        "grpc-port",
			EnvVars:     util.EnvVars("grpc-port"),
			Usage:       "grpc server port, -1 for disable the grpc server",
			Value:       -1,
			Destination: &conf.Server.GrpcPort,
		},
		&cli.StringFlag{
			Name:    "grpc-servers",
			EnvVars: util.EnvVars("grpc-servers"),
			Usage:   "upstream servers, for proxy mode(grpc address and port), use comma for split",
		},
		&cli.StringFlag{
			Name:        "grpc-auth",
			EnvVars:     util.EnvVars("grpc-auth"),
			Usage:       "grpc auth token",
			Value:       "password",
			Destination: &conf.Backend.GRPC.Auth,
		},
		&cli.StringFlag{
			Name:        "grpc-proxy",
			EnvVars:     util.EnvVars("grpc-proxy"),
			Usage:       "grpc proxy address, in the format of http://127.0.0.1:8080 or socks5://127.0.0.1:1080",
			Value:       "",
			Destination: &conf.Backend.GRPC.Proxy,
		},
		&cli.StringFlag{
			Name:    "idle-time",
			EnvVars: util.EnvVars("idle-time"),
			Usage:   "time out of an idle connection",
		},
		&cli.BoolFlag{
			Name:        "control-all",
			Aliases:     []string{"ctl-a"},
			EnvVars:     util.EnvVars("ctl-a"),
			Usage:       "enable container control",
			Destination: &conf.Server.Control.All,
		},
		&cli.BoolFlag{
			Name:        "control-start",
			Aliases:     []string{"ctl-s"},
			EnvVars:     util.EnvVars("ctl-s"),
			Usage:       "enable container start  ",
			Destination: &conf.Server.Control.Start,
		},
		&cli.BoolFlag{
			Name:        "control-stop",
			Aliases:     []string{"ctl-t"},
			EnvVars:     util.EnvVars("ctl-t"),
			Usage:       "enable container stop   ",
			Destination: &conf.Server.Control.Stop,
		},
		&cli.BoolFlag{
			Name:        "control-restart",
			Aliases:     []string{"ctl-r"},
			EnvVars:     util.EnvVars("ctl-r"),
			Usage:       "enable container restart",
			Destination: &conf.Server.Control.Restart,
		},
		&cli.BoolFlag{
			Name:        "enable-share",
			Aliases:     []string{"share"},
			EnvVars:     util.EnvVars("share"),
			Usage:       "enable share the container's terminal",
			Destination: &conf.Server.EnableShare,
		},
		&cli.BoolFlag{
			Name:        "enable-audit",
			Aliases:     []string{"audit"},
			EnvVars:     util.EnvVars("audit"),
			Usage:       "enable audit the container outputs",
			Destination: &conf.Server.EnableAudit,
		},
		&cli.StringFlag{
			Name:        "audit-dir",
			EnvVars:     util.EnvVars("audit-dir"),
			Value:       "audit",
			Usage:       "container audit log dir path",
			Destination: &conf.Server.AuditLogDir,
		},
		&cli.BoolFlag{
			Name:    "help",
			Aliases: []string{"h"},
			Usage:   "show help",
		},
	}

	sort.Sort(cli.FlagsByName(appFlags))

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
			// parse idleTime
			t := c.String("idle-time")
			idleTime, err := time.ParseDuration(t)
			if err != nil && t != "" {
				logrus.Fatalf("parse idle-time error: %s", err)
			} else {
				conf.Server.IdleTime = idleTime
			}

			// defaultArgs := "-e HISTCONTROL=ignoredups -e TERM=xterm"

			ctl := conf.Server.Control
			if ctl.Start || ctl.Stop || ctl.Restart || ctl.All {
				conf.Server.Control.Enable = true
			}

			servers := strings.Split(c.String("grpc-servers"), ",")
			if servers[0] != "" {
				conf.Backend.GRPC.Servers = servers
			}
			if conf.Debug {
				logrus.SetLevel(logrus.DebugLevel)
			} else {
				gin.SetMode(gin.ReleaseMode)
			}
			logrus.Debugf("got config: %+v", conf)

			run(c, *conf)
			return nil
		},
	}

	app.Run(os.Args)
}
