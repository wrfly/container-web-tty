package route

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/yudai/gotty/webtty"
)

func (server *Server) handleShare(c *gin.Context) {
	ctx := c.Request.Context()
	cid := c.Param("id")
	cInfo := server.containerCli.GetInfo(ctx, cid)

	conn, err := server.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("upgrade ws error: %s", err)
		return
	}
	defer conn.Close()

	// note: must read the init message
	// although it's useless in this situation
	server.readInitMessage(conn)

	server.mMux.RLock()
	masterTTY, ok := server.masters[cInfo.ID]
	server.mMux.RUnlock()
	if !ok {
		log.Error("share terminal error, master not found")
		conn.WriteMessage(websocket.CloseMessage, []byte("master container not found, exit"))
		return
	}

	titleBuf, err := server.makeTitleBuff(cInfo)
	if err != nil {
		e := fmt.Sprintf("failed to fill window title template: %s", err)
		conn.WriteMessage(websocket.CloseMessage, []byte(e))
		log.Error(e)
		return
	}

	fork := masterTTY.Fork(ctx, server.options.Collaborate)
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

	if err := tty.Run(ctx); err != nil && err != webtty.ErrMasterClosed {
		e := fmt.Sprintf("failed to run webtty: %s", err)
		log.Error(e)
	}
}
