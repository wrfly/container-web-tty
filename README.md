# Container web TTY

Tired of typing `docker ps | grep xxx` && `docker exec -ti xxxx sh` ? Try me!

Although I like terminal, I still want a better tool to get inside of the containers and do some things. So I build this `container-web-tty`. It can help you get into the container and execute commands via a web-tty, based on [yudai/gotty](https://github.com/yudai/gotty) with some change.

Both `docker` and `kubectl` are supported.

## TODO

- [x] it works
- [x] docker backend
- [ ] kubectl backend
- [x] beautiful index
- [ ] proxy mode(client -> server's containers)

~~- [ ] session and auth~~

~~- [ ] hold the TTY(with timeout)~~

## Show-off

List the containers on your machine:
![List containers](images/list.png)

It will execute `sh` if there is no `/bin/bash` inside the container:
![sh by dfault](images/sh.png)

Of Cause `bash`:
![also bash](images/bash.png)