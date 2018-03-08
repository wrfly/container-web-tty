# Container web TTY

Tired of typing `docker ps | grep xxx` && `docker exec -ti xxxx sh` ? Try me!

Although I like terminal, I still want a better tool to get inside of the containers and do some things. So I build this `container-web-tty`. It can help you get into the container and execute commands via a web-tty, based on [yudai/gotty](https://github.com/yudai/gotty) with a little change.

Both `docker` and `kubectl` are supported.

## TODO

- [ ] it works
- [ ] docker backend
- [ ] kubectl backend
- [ ] beautiful index
- [ ] hold the TTY(with timeout)
- [ ] proxy mode(client -> server's containers)

~~- [ ] session and auth~~
