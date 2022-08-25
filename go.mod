module github.com/wrfly/container-web-tty

go 1.16

require (
	github.com/containerd/containerd v1.6.8 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker v20.10.8+incompatible
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a
	github.com/gin-gonic/gin v1.7.3
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/websocket v1.4.2
	github.com/sirupsen/logrus v1.8.1
	github.com/urfave/cli/v2 v2.3.0
	github.com/wrfly/ecp v0.1.1-0.20190725160759-97269b9e95f0
	github.com/wrfly/pubsub v0.0.0-20200314104228-47828c5578b6
	github.com/yudai/gotty v2.0.0-alpha.3+incompatible
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	google.golang.org/grpc v1.43.0
	k8s.io/api v0.23.6
	k8s.io/apimachinery v0.23.6
	k8s.io/client-go v0.23.6
)
