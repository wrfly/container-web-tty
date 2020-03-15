package route

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/wrfly/container-web-tty/types"
	"github.com/yudai/gotty/webtty"
)

func (server *Server) handleShare(c *gin.Context) {
	execID := c.Param("id")
	server.m.RLock()
	masterTTY, ok := server.masters[execID]
	server.m.RUnlock()
	if !ok || masterTTY == nil {
		c.String(http.StatusBadRequest, "share terminal error, master not found")
		return
	}

	server.processShare(c, execID, masterTTY, server.options.Collaborate)
}

func (server *Server) processShare(c *gin.Context, execID string, masterTTY *types.MasterTTY,
	collaborate bool) {
	conn, err := server.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("upgrade ws error: %s", err)
		return
	}
	defer conn.Close()
	// note: must read the init message
	// although it's useless in this situation
	server.readInitMessage(conn)

	ctx := c.Request.Context()
	containerID, ok := server.getContainerID(execID)
	if !ok {
		log.Error("share terminal error, exec not found")
		conn.WriteMessage(websocket.CloseMessage,
			[]byte("exec container not found, exit"))
		return
	}

	cInfo := server.containerCli.GetInfo(ctx, containerID)
	titleBuf, err := server.makeTitleBuff(cInfo)
	if err != nil {
		e := fmt.Sprintf("failed to fill window title template: %s", err)
		conn.WriteMessage(websocket.CloseMessage, []byte(e))
		log.Error(e)
		return
	}

	fork := masterTTY.Fork(ctx, collaborate)
	defer fork.Close()

	tty, err := webtty.New(
		&wsWrapper{conn},
		newSlave(fork, true),
		[]webtty.Option{
			webtty.WithWindowTitle(titleBuf),
			webtty.WithPermitWrite()}...,
	)
	if err != nil {
		e := fmt.Sprintf("failed to create webtty: %s", err)
		conn.WriteMessage(websocket.CloseMessage, []byte(e))
		log.Error(e)
		return
	}

	err = tty.Run(ctx)
	if err != nil && err != webtty.ErrMasterClosed {
		e := fmt.Sprintf("failed to run webtty: %s", err)
		log.Error(e)
	}
}
