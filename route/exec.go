package route

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/wrfly/container-web-tty/audit"
	"github.com/wrfly/container-web-tty/types"
	"github.com/yudai/gotty/webtty"
)

func (server *Server) handleExec(c *gin.Context, counter *counter) {
	cInfo := server.containerCli.GetInfo(c.Request.Context(), c.Param("id"))
	server.generateHandleWS(c.Request.Context(), counter, cInfo).
		ServeHTTP(c.Writer, c.Request)
}

func (server *Server) generateHandleWS(ctx context.Context, counter *counter, container types.Container) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if container.Shell == "" {
			log.Errorf("cannot find a valid shell in container [%s]", container.ID)
			return
		}

		num := counter.add(1)
		closeReason := "unknown reason"

		defer func() {
			num := counter.done()
			if strings.Contains(closeReason, "error") {
				log.Errorf("Connection closed by %s: %s, connections: %d",
					closeReason, r.RemoteAddr, num)
			}
			log.Infof("Connection closed by %s: %s, connections: %d",
				closeReason, r.RemoteAddr, num)
		}()

		if int64(server.options.MaxConnection) != 0 {
			if num > server.options.MaxConnection {
				closeReason = "exceeding max number of connections"
				return
			}
		}

		log.Infof("New client connected: %s, connections: %d", r.RemoteAddr, num)

		conn, err := server.upgrader.Upgrade(w, r, nil)
		if err != nil {
			closeReason = err.Error()
			return
		}
		defer conn.Close()

		cctx, timeoutCancel := context.WithCancel(ctx)
		defer timeoutCancel()

		err = server.processTTY(cctx, timeoutCancel, conn, container)
		switch err {
		case ctx.Err():
			closeReason = "cancelation"
		case cctx.Err():
			closeReason = "time out"
		case webtty.ErrSlaveClosed:
			closeReason = "backend closed"
		case webtty.ErrMasterClosed:
			closeReason = "tab closed"
		default:
			closeReason = fmt.Sprintf("an error: %s", err)
		}
	}
}

func (server *Server) processTTY(ctx context.Context, timeoutCancel context.CancelFunc,
	conn *websocket.Conn, container types.Container) error {
	arguments, err := server.readInitMessage(conn)
	if err != nil {
		return err
	}
	log.Debugf("exec container: %s, params: %s", container.ID, arguments)

	q, err := parseQuery(strings.TrimSpace(arguments))
	if err != nil {
		return err
	}
	container.Exec = types.ExecOptions{
		Cmd:        q.Get("cmd"),
		Env:        q.Get("env"),
		User:       q.Get("user"),
		Privileged: q.Get("p") != "",
	}

	containerTTY, err := server.containerCli.Exec(ctx, container)
	if err != nil {
		return fmt.Errorf("exec container error: %s", err)
	}
	defer containerTTY.Exit()

	// handle timeout
	tout := server.options.IdleTime
	if tout.Seconds() != 0 {
		go func() {
			timer := time.NewTimer(tout)
			activeChan := containerTTY.ActiveChan()
			for {
				select {
				case <-timer.C:
					timer.Stop()
					timeoutCancel()
					return
				case <-activeChan:
					// the connection is active, reset the timer
					timer.Reset(tout)
				}
			}

		}()
	}

	titleBuf, err := server.makeTitleBuff(container)
	if err != nil {
		return fmt.Errorf("failed to fill window title template: %s", err)
	}

	opts := []webtty.Option{
		webtty.WithWindowTitle(titleBuf),
		webtty.WithPermitWrite(),
		// webtty.WithReconnect(10), // not work....
	}

	wrapper := &wsWrapper{conn}
	masterTTY, err := types.NewMasterTTY(ctx, containerTTY, container.ID)
	if err != nil {
		return err
	}
	server.mMux.Lock()
	if _, ok := server.masters[container.ID]; !ok {
		server.masters[container.ID] = masterTTY
	}
	server.mMux.Unlock()
	defer func() {
		server.mMux.Lock()
		delete(server.masters, container.ID)
		server.mMux.Unlock()
	}()

	if server.options.EnableAudit {
		go audit.LogTo(ctx, masterTTY.Fork(ctx), audit.LogOpts{
			Dir:         server.options.AuditLogDir,
			ContainerID: container.ID,
			ClientIP:    conn.RemoteAddr().String(),
		})
	}

	log.Infof("new web tty for container: %s", container.ID)
	tty, err := webtty.New(wrapper, masterTTY, opts...)
	if err != nil {
		return fmt.Errorf("failed to create webtty: %s", err)
	}

	return tty.Run(ctx)
}
