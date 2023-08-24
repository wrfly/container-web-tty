package route

import (
	"fmt"

	"github.com/wrfly/container-web-tty/third-part/gotty/webtty"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/wrfly/container-web-tty/types"
)

func (server *Server) processShare(c *gin.Context, execID string, masterTTY *types.MasterTTY) {
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
	var titleExtra = "[READONLY]"
	if server.options.Collaborate {
		titleExtra = "[SLAVE]"
	}
	titleBuf, err := server.makeTitleBuff(cInfo, titleExtra)
	if err != nil {
		e := fmt.Sprintf("failed to fill window title template: %s", err)
		conn.WriteMessage(websocket.CloseMessage, []byte(e))
		log.Error(e)
		return
	}

	master := masterTTY.Fork(ctx, true)
	defer master.Close()

	ttyOptions := []webtty.Option{webtty.WithWindowTitle(titleBuf)}
	if server.options.Collaborate {
		ttyOptions = append(ttyOptions, webtty.WithPermitWrite())
	}

	tty, err := webtty.New(
		&wsWrapper{conn},
		newSlave(master),
		ttyOptions...,
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
