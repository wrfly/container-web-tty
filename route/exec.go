package route

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/wrfly/container-web-tty/third-part/gotty/webtty"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/wrfly/container-web-tty/audit"
	"github.com/wrfly/container-web-tty/types"
)

func (server *Server) handleExecRedirect(c *gin.Context) {
	containerID := c.Param("cid")
	execID := server.setContainerID(containerID)
	base := filepath.Join(server.options.Base, "/exec/") + "/"
	if query := c.Request.URL.RawQuery; query != "" {
		c.Redirect(302, base+execID+"?"+c.Request.URL.RawQuery)
	} else {
		c.Redirect(302, base+execID)
	}
}

func (server *Server) handleExec(c *gin.Context, counter *counter) {
	execID := c.Param("eid")
	containerID, ok := server.getContainerID(execID)
	if !ok {
		c.String(http.StatusBadRequest, fmt.Sprintf("exec id %s not found", execID))
		return
	}

	server.m.RLock()
	masterTTY, ok := server.masters[execID]
	server.m.RUnlock()
	if ok { // exec ID exist, use the same master
		log.Infof("using exist master for exec %s", execID)
		server.processShare(c, execID, masterTTY)
		return
	}

	cInfo := server.containerCli.GetInfo(c.Request.Context(), containerID)
	server.generateHandleWS(c.Request.Context(), execID, counter, cInfo).
		ServeHTTP(c.Writer, c.Request)
}

func (server *Server) generateHandleWS(ctx context.Context, execID string, counter *counter, container types.Container) http.HandlerFunc {
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

		err = server.processTTY(cctx, execID, timeoutCancel, conn, container)
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

func (server *Server) processTTY(ctx context.Context, execID string, timeoutCancel context.CancelFunc,
	conn *websocket.Conn, container types.Container) error {
	arguments, err := server.readInitMessage(conn)
	if err != nil {
		return err
	}
	log.Debugf("exec container: %s, params: [%s]", container.ID[:7], arguments)

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
	defer func() {
		log.Infof("container %s exit", container.ID[:7])
		if err := containerTTY.Exit(); err != nil {
			log.Warnf("exit container err: %s", err)
		}
	}()

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
				case _, ok := <-activeChan:
					if !ok {
						return
					}
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
	}

	shareID := fmt.Sprintf("%s-%d", container.ID, time.Now().UnixNano())
	masterTTY, err := types.NewMasterTTY(ctx, containerTTY, shareID)
	if err != nil {
		return err
	}

	server.m.Lock()
	server.masters[execID] = masterTTY
	server.m.Unlock()

	defer func() {
		// if master dead, all slaves dead
		server.m.Lock()
		masterTTY.Close()
		delete(server.masters, execID)
		delete(server.execs, execID)
		server.m.Unlock()
	}()

	if server.options.EnableAudit {
		go audit.LogTo(ctx, masterTTY.Fork(ctx, false), audit.LogOpts{
			Dir:         server.options.AuditLogDir,
			ContainerID: container.ID,
			ClientIP:    conn.RemoteAddr().String(),
		})
	}

	log.Infof("new web tty for container: %s", container.ID[:7])
	wrapper := &wsWrapper{conn}
	tty, err := webtty.New(wrapper, masterTTY, opts...)
	if err != nil {
		return fmt.Errorf("failed to create webtty: %s", err)
	}

	return tty.Run(ctx)
}
