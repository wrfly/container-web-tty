# Container web TTY

[![Go Report Card](https://goreportcard.com/badge/github.com/wrfly/container-web-tty)](https://goreportcard.com/report/github.com/wrfly/container-web-tty)
[![Master Build Status](https://travis-ci.org/wrfly/container-web-tty.svg?branch=master)](https://travis-ci.org/wrfly/container-web-tty)
[![GoDoc](https://godoc.org/github.com/wrfly/container-web-tty?status.svg)](https://godoc.org/github.com/wrfly/container-web-tty)
[![license](https://img.shields.io/github/license/wrfly/container-web-tty.svg)](https://github.com/wrfly/container-web-tty/blob/master/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/wrfly/container-web-tty.svg)](https://hub.docker.com/r/wrfly/container-web-tty)
[![MicroBadger Size](https://img.shields.io/microbadger/image-size/wrfly/container-web-tty.svg)](https://hub.docker.com/r/wrfly/container-web-tty)
[![GitHub release](https://img.shields.io/github/release/wrfly/container-web-tty.svg)](https://github.com/wrfly/container-web-tty/releases)
[![Github All Releases](https://img.shields.io/github/downloads/wrfly/container-web-tty/total.svg)](https://github.com/wrfly/container-web-tty/releases)

[中文](README.ZH.md)

Tired of typing `docker ps | grep xxx` && `docker exec -ti xxxx sh` ? Try me!

Although I like terminal, I still want a better tool to get into the containers to do some debugging or checking.
So I build this `container-web-tty`. It can help you get into the container and execute commands via a web-tty,
based on [yudai/gotty](https://github.com/yudai/gotty) with some changes.

Both `docker` and `kubectl` are supported.

## Usage

Of cause you can run it by downloading the binary, but thare are some
`Copy-and-Paste` ways.

### Using docker

You can start `container-web-tty` inside a container by mounting `docker.sock`:

```bash
docker run --rm -ti --name web-tty \
    -p 8080:8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    wrfly/container-web-tty
```

### Using kubernetes

Or you can mount the kubernetes config file:

```bash
docker run --rm -ti --name web-tty \
    -p 8080:8080 \
    -e WEB_TTY_BACKEND=kube \
    -e WEB_TTY_KUBE_CONFIG=/kube.config \
    -v ~/.kube/config:/kube.config \
    wrfly/container-web-tty
```

### Using local <-> remote (gRPC)

You can deploy `container-web-tty` in remote servers, and connect
to it via a local `container-web-tty`. They use gRPC for communication.

This is useful when you cannot get the remote servers or there are more
than one server that you need to connect to.

#### Remote

Host `192.168.66.1` and `192.168.66.2` both running:

```bash
docker run --rm -ti --name web-tty \
    -p 8080:8080 \
    -p 8090:8090 \
    -e WEB_TTY_GRPC_PORT=8090 \
    -e WEB_TTY_GRPC_AUTH=96ssW0rd \
    -v /var/run/docker.sock:/var/run/docker.sock \
    wrfly/container-web-tty
```

Notes:

- You can disable the HTTP server by setting `WEB_TTY_PORT=-1`
- The `WEB_TTY_GRPC_AUTH` must be the same between all hosts

#### Local

```bash
docker run --rm -ti --name web-tty \
    -p 8080:8080 \
    -e WEB_TTY_BACKEND=grpc \
    -e WEB_TTY_GRPC_AUTH=96ssW0rd \
    -e WEB_TTY_GRPC_SERVERS=192.168.66.1:8090,192.168.66.2:8090 \
    wrfly/container-web-tty
```

Now you will see all the containers of all the servers via *<http://localhost:8080>*

## Keyboard Shortcuts (Linux)

- Cut the word before the cursor `Ctrl+w` => **You cannot do it for now** (I'll working on it for `Ctrl+Backspace`, but I know little about js)
- Copy:  `Ctrl+Shift+c` => `Ctrl+Insert`
- Paste: `Ctrl+Shift+v` => `Shift+Insert`

## Features

- [x] it works
- [x] docker backend
- [x] kubectl backend
- [x] beautiful index
- [x] start|stop|restart container(docker backend only)
- [x] environment injection (extra params)
- [x] proxy mode (client -> server's containers)
- [x] auth(only in proxy mode)
- [x] TTY timeout (idle timeout)
- [ ] history audit
- [x] container logs (just click the container name)
- [x] exec arguments (append an extra "?cmd=xxx" argument to exec URL)

## Show-off

List the containers on your machine:

![list](images/list.png)

It will execute `/bin/sh` if there is no `/bin/bash` inside the container:

<img src="images/sh.png" width="400" height="150">

`/bin/bash`:

<img src="images/bash.png" width="400" height="150">

Run custom command:

<img src="images/cmd.png" width="400" height="150">

Get container logs:

![logs](images/logs.png)