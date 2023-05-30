FROM alpine
COPY bin/container-web-tty /usr/bin
EXPOSE 8080
CMD [ "container-web-tty" ]
