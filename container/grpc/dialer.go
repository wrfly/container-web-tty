package grpc

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"google.golang.org/grpc"
)

const (
	pingTimeout  = time.Second
	testDialAddr = "ipinfo.io:80"
)

type dialer struct {
	socksD proxy.Dialer
	proxyD dialFunc
}

func (d *dialer) Dial(network, addr string) (net.Conn, error) {
	if d.socksD != nil {
		return d.socksD.Dial(network, addr)
	}
	return d.proxyD(network, addr)
}

type dialFunc func(network, address string) (net.Conn, error)

func newDialOption(proxyStr string) (grpc.DialOption, error) {
	u, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}

	var grpcDialer func(addr string, t time.Duration) (net.Conn, error)

	switch u.Scheme {
	case "socks5":
		addr := fmt.Sprintf("%s", u.Host)
		auth := &proxy.Auth{}
		if user := u.User; user != nil {
			auth.User = user.Username()
			auth.Password, _ = user.Password()
		}
		socksDialer, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}

		if _, err := dialWithTimeout(&dialer{socksD: socksDialer},
			testDialAddr, pingTimeout); err != nil {
			return nil, err
		}
		grpcDialer = func(addr string, t time.Duration) (net.Conn, error) {
			return dialWithTimeout(&dialer{socksD: socksDialer}, addr, t)
		}
	case "http", "https":
		p := goproxy.NewProxyHttpServer()
		p.Logger.SetOutput(logrus.StandardLogger().Out)
		httpDialer := p.NewConnectDialToProxy(proxyStr)

		if _, err := dialWithTimeout(&dialer{proxyD: httpDialer},
			testDialAddr, pingTimeout); err != nil {
			return nil, err
		}

		grpcDialer = func(addr string, t time.Duration) (net.Conn, error) {
			return dialWithTimeout(&dialer{proxyD: httpDialer}, addr, t)
		}
	default:
		return nil, fmt.Errorf("unsupport scheme %s", u.Scheme)
	}

	return grpc.WithDialer(grpcDialer), nil
}

func dialWithTimeout(dialer proxy.Dialer, addr string,
	timeout time.Duration) (net.Conn, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	connChan := make(chan net.Conn)

	go func() {
		conn, err := dialer.Dial("tcp", addr)
		if err == nil {
			connChan <- conn
			return
		}
		logrus.Errorf("dial to %s error: %s", addr, err)
	}()

	select {
	case <-timer.C:
		return nil, fmt.Errorf("dial %s timeout (%s)", addr, timeout)
	case c := <-connChan:
		return c, nil
	}
}
