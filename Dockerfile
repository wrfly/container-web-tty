FROM alpine
RUN [ ! -e /etc/nsswitch.conf ] && echo 'hosts: files dns' > /etc/nsswitch.conf
COPY bin/container-web-tty /usr/bin
EXPOSE 8080
CMD [ "container-web-tty" ]
