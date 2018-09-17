# Container web TTY

[![Go Report Card](https://goreportcard.com/badge/github.com/wrfly/container-web-tty)](https://goreportcard.com/report/github.com/wrfly/container-web-tty)
[![Master Build Status](https://travis-ci.org/wrfly/container-web-tty.svg?branch=master)](https://travis-ci.org/wrfly/container-web-tty)
[![GoDoc](https://godoc.org/github.com/wrfly/container-web-tty?status.svg)](https://godoc.org/github.com/wrfly/container-web-tty)
[![license](https://img.shields.io/github/license/wrfly/container-web-tty.svg)](https://github.com/wrfly/container-web-tty/blob/master/LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/wrfly/container-web-tty.svg)](https://hub.docker.com/r/wrfly/container-web-tty)
[![MicroBadger Size](https://img.shields.io/microbadger/image-size/wrfly/container-web-tty.svg)](https://hub.docker.com/r/wrfly/container-web-tty)
[![GitHub release](https://img.shields.io/github/release/wrfly/container-web-tty.svg)](https://github.com/wrfly/container-web-tty/releases)
[![Github All Releases](https://img.shields.io/github/downloads/wrfly/container-web-tty/total.svg)](https://github.com/wrfly/container-web-tty/releases)

[English](README.md)

当我们想进入某个容器内部的时候，通常会执行这个命令组合 `docker ps | grep xxx` && `docker exec -ti xxxx sh`，
但老这样敲也是很烦，也许你可以试一下这个项目。

虽然我很喜欢终端，但是我仍然希望有一个更好的工具来进入容器内部去做一些检查或者debug。所以我写了这个项目，它能够帮助你通过点击网页的方式
进到容器里执行命令。初版的代码是基于[yudai/gotty](https://github.com/yudai/gotty)这个项目的，感谢yudai。

后端可以对接docker或者kubectl。

## 使用

你可以从release页面下载二进制运行这个程序，但这里有一些“复制粘贴”的方法。

### 通过 docker

把`docker.sock`挂在到容器里就完事儿了

```bash
docker run --rm -ti --name web-tty \
    -p 8080:8080 \
    -v /var/run/docker.sock:/var/run/docker.sock \
    wrfly/container-web-tty
```

### 通过 kubernetes

你需要把kubernetes的配置文件挂进去，默认是在 `$HOME/.kube/config`，然后指定一下backed的类型，也就是`kube`

```bash
docker run --rm -ti --name web-tty \
    -p 8080:8080 \
    -e WEB_TTY_BACKEND=kube \
    -e WEB_TTY_KUBE_CONFIG=/kube.config \
    -v ~/.kube/config:/kube.config \
    wrfly/container-web-tty
```

### 通过 gRPC （代理模式）

当我们有很多server需要接入的时候，就可以使用这种模式把远程的`container-web-tty`们
**merge** 到一起，典型的CS模式，通过gRPC通信。

在一个界面上查看多台机器上的容器。

#### 远程配置

假如有两台机器 `192.168.66.1` 和 `192.168.66.2`，他们可以用如下的命令来启动`container-web-tty`

```bash
docker run --rm -ti --name web-tty \
    -p 8080:8080 \
    -p 8090:8090 \
    -e WEB_TTY_GRPC_PORT=8090 \
    -e WEB_TTY_GRPC_AUTH=96ssW0rd \
    -v /var/run/docker.sock:/var/run/docker.sock \
    wrfly/container-web-tty
```

注意:

- 你可以通过设置 `WEB_TTY_PORT=-1` 的方式来关闭HTTPserver，拒绝一般接入
- 这个 `WEB_TTY_GRPC_AUTH` key 在所有机器上必须要相同（目前）

#### 本地配置

```bash
docker run --rm -ti --name web-tty \
    -p 8080:8080 \
    -e WEB_TTY_BACKEND=grpc \
    -e WEB_TTY_GRPC_AUTH=96ssW0rd \
    -e WEB_TTY_GRPC_SERVERS=192.168.66.1:8090,192.168.66.2:8090 \
    wrfly/container-web-tty
```

现在你就可以通过访问 *<http://localhost:8080>* 来获取两台机器上所有的容器

## 快捷键 (Linux)

- Cut the word before the cursor `Ctrl+w` => **You cannot do it for now** (I'll working on it for `Ctrl+Backspace`, but I know little about js)
- Copy:  `Ctrl+Shift+c` => `Ctrl+Insert`
- Paste: `Ctrl+Shift+v` => `Shift+Insert`

## 特性

- [x] 能用了
- [x] 对接 docker 后端
- [x] 对接 kubectl 的后端
- [x] 比较好看的前端界面
- [x] start|stop|restart container(docker backend only)
- [x] 参数注入，比如环境变量
- [x] 代理模式 (本地连接到远程机器上的容器)
- [x] 认证（仅限代理模式）
- [x] 超时自动断开
- [ ] 历史记录审计
- [x] 容器日志
- [x] 自定义执行命令

## 效果展示

列出所有容器:

![list](images/list.png)

在选择shell的时候，优选选择bash，如果没有，就依次选择ash，sh，再没有就退出了。

`/bin/sh`:

<img src="images/sh.png" width="400" height="150">

`/bin/bash`:

<img src="images/bash.png" width="400" height="150">

运行指定命令:

<img src="images/cmd.png" width="400" height="150">

查看容器日志:

![logs](images/logs.png)