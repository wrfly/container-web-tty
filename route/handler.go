package route

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/yudai/gotty/webtty"

	"github.com/wrfly/container-web-tty/types"
)

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
			log.Infof(
				"connection closed by %s: %s, connections: %d",
				closeReason, r.RemoteAddr, num,
			)
		}()

		// if int64(server.options.MaxConnection) != 0 {
		// 	if num > server.options.MaxConnection {
		// 		closeReason = "exceeding max number of connections"
		// 		return
		// 	}
		// }

		log.Infof("new client connected: %s, connections: %d", r.RemoteAddr, num)

		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return
		}

		conn, err := server.upgrader.Upgrade(w, r, nil)
		if err != nil {
			closeReason = err.Error()
			return
		}
		defer conn.Close()

		err = server.processWSConn(ctx, conn, container)
		switch err {
		case ctx.Err():
			closeReason = "cancelation"
		case webtty.ErrSlaveClosed:
			closeReason = "backend closed"
		case webtty.ErrMasterClosed:
			closeReason = "tab closed"
		default:
			closeReason = fmt.Sprintf("an error: %s", err)
		}
	}
}

func (server *Server) processWSConn(ctx context.Context, conn *websocket.Conn, container types.Container) error {
	typ, initLine, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to authenticate websocket connection")
	}
	if typ != websocket.TextMessage {
		return fmt.Errorf("failed to authenticate websocket connection: invalid message type")
	}

	var init types.InitMessage
	err = json.Unmarshal(initLine, &init)
	if err != nil {
		return fmt.Errorf("failed to authenticate websocket connection")
	}
	// if init.AuthToken != server.options.Credential {
	// 	return fmt.Errorf("failed to authenticate websocket connection")
	// }

	log.Debugf("exec container: %s", container.ID)
	containerTTY, err := server.containerCli.Exec(ctx, container)
	if err != nil {
		return fmt.Errorf("exec container error: %s", err)
	}
	defer containerTTY.Exit()

	cIP := "127.0.0.1"
	if len(container.IPs) > 0 {
		cIP = container.IPs[0]
	}

	titleVars := server.titleVariables(
		[]string{"server"},
		map[string]map[string]interface{}{
			"server": map[string]interface{}{
				"containerIP":   cIP,
				"containerName": container.Name,
				"containerID":   container.ID,
			},
		},
	)

	titleBuf := new(bytes.Buffer)
	err = titleTemplate.Execute(titleBuf, titleVars)
	if err != nil {
		return fmt.Errorf("failed to fill window title template: %s", err)
	}

	opts := []webtty.Option{
		webtty.WithWindowTitle(titleBuf.Bytes()),
		webtty.WithPermitWrite(),
	}

	wrapper := &wsWrapper{conn}
	tty, err := webtty.New(wrapper, containerTTY, opts...)
	if err != nil {
		return fmt.Errorf("failed to create webtty: %s", err)
	}

	err = tty.Run(ctx)

	return err
}

func (server *Server) handleExec(c *gin.Context, cInfo types.Container) {
	titleVars := server.titleVariables(
		[]string{"server"},
		map[string]map[string]interface{}{
			"server": map[string]interface{}{
				"containerName": cInfo.Name,
				"containerID":   cInfo.ID,
			},
		},
	)

	titleBuf := new(bytes.Buffer)
	err := titleTemplate.Execute(titleBuf, titleVars)
	if err != nil {
		c.Error(err)
	}

	indexVars := map[string]interface{}{
		"title": titleBuf.String(),
	}

	indexBuf := new(bytes.Buffer)
	err = indexTemplate.Execute(indexBuf, indexVars)
	if err != nil {
		c.Error(err)
	}

	c.Writer.Write(indexBuf.Bytes())
}

func (server *Server) handleAuthToken(c *gin.Context) {
	c.Header("Content-Type", "application/javascript")
	// @TODO hashing?
	c.String(200, "var gotty_auth_token = '%s';", server.options.Credential)
}

func (server *Server) handleConfig(c *gin.Context) {
	c.Header("Content-Type", "application/javascript")
	c.String(200, "var gotty_term = '%s';", server.options.Term)
}

// titleVariables merges maps in a specified order.
// varUnits are name-keyed maps, whose names will be iterated using order.
func (server *Server) titleVariables(order []string, varUnits map[string]map[string]interface{}) map[string]interface{} {
	titleVars := map[string]interface{}{}

	for _, name := range order {
		vars, ok := varUnits[name]
		if !ok {
			panic("title variable name error")
		}
		for key, val := range vars {
			titleVars[key] = val
		}
	}

	// safe net for conflicted keys
	for _, name := range order {
		titleVars[name] = varUnits[name]
	}

	return titleVars
}

func (server *Server) handleListContainers(c *gin.Context) {
	listVars := map[string]interface{}{
		"title":      "List Containers",
		"containers": server.containerCli.List(c.Request.Context()),
		"control":    server.options.Control,
	}

	listBuf := new(bytes.Buffer)
	err := listTemplate.Execute(listBuf, listVars)
	if err != nil {
		c.Error(err)
	}

	c.Writer.Write(listBuf.Bytes())
}

func (server *Server) handleContainerActions(c *gin.Context, action string) {
	cid := c.Param("id")
	log.Debugf("client [%s] is going to [%s] container [%s]",
		c.ClientIP(), action, cid)
	var err error
	switch action {
	case "start":
		err = server.containerCli.Start(c.Request.Context(), cid)
	case "stop":
		err = server.containerCli.Stop(c.Request.Context(), cid)
	case "restart":
		err = server.containerCli.Restart(c.Request.Context(), cid)
	}
	if err != nil {
		c.JSON(500, types.ContainerActionMessage{
			Code:  500,
			Error: err.Error(),
		})
		return
	}
	c.JSON(0, types.ContainerActionMessage{
		Code:    0,
		Message: fmt.Sprintf("%s container %s successfully", action, cid[:7]),
	})
}

func (server *Server) handleStartContainer(c *gin.Context) {
	server.handleContainerActions(c, "start")
}

func (server *Server) handleStopContainer(c *gin.Context) {
	server.handleContainerActions(c, "stop")
}

func (server *Server) handleRestartContainer(c *gin.Context) {
	server.handleContainerActions(c, "restart")
}
