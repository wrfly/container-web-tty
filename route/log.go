package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/yudai/gotty/webtty"

	"github.com/wrfly/container-web-tty/types"
	"github.com/wrfly/container-web-tty/util"
)

func (server *Server) handleLogs(c *gin.Context) {
	ctx := c.Request.Context()

	conn, err := server.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "server error: %s", err)
		return
	}
	defer conn.Close()

	initArg, err := server.readInitMessage(conn)
	if err != nil {
		c.String(http.StatusBadRequest, "read init message error: %s", err)
		return
	}

	q, err := parseQuery(initArg)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	follow := true
	if v := q.Get("follow"); v != "1" && v != "" {
		follow = false
	}
	tail := "10"
	if v := q.Get("tail"); v != "" {
		tail = v
	}
	opts := types.LogOptions{
		ID:     c.Param("cid"),
		Follow: follow,
		Tail:   tail,
	}

	container := server.containerCli.GetInfo(ctx, opts.ID)

	log.Debugf("get logs of container: %s", container.ID)
	logsReadCloser, err := server.containerCli.Logs(ctx, opts)
	if err != nil {
		c.String(http.StatusInternalServerError, "get logs error: %s", err)
		return
	}
	defer logsReadCloser.Close()

	titleBuf, err := server.makeTitleBuff(container)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to fill window title template: %s", err)
		return
	}

	tty, err := webtty.New(
		&wsWrapper{conn},
		newSlave(util.NopRWCloser(logsReadCloser), false),
		[]webtty.Option{
			webtty.WithWindowTitle(titleBuf),
			webtty.WithPermitWrite(), // can type "enter"
		}...,
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create webtty: %s", err)
		return
	}

	if err := tty.Run(ctx); err != nil {
		if err != webtty.ErrMasterClosed && err != webtty.ErrSlaveClosed {
			log.Errorf("failed to run webtty: %s", err)
		}
	}
}
