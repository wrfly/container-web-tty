package route

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"

	"github.com/wrfly/container-web-tty/types"
)

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
		"loc":        server.options.ShowLocation,
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

func (server *Server) makeTitleBuff(c types.Container, extra ...string) ([]byte, error) {
	location := "localhost"
	if c.LocServer != "" {
		location = c.LocServer
	}

	cName := c.Name
	if len(extra) != 0 {
		cName = extra[0] + " " + c.Name
	}

	titleVars := server.titleVariables(
		[]string{"server"},
		map[string]map[string]interface{}{
			"server": map[string]interface{}{
				"containerLoc":  location,
				"containerName": cName,
				"containerID":   c.ID,
			},
		},
	)
	titleBuf := new(bytes.Buffer)
	if err := titleTemplate.Execute(titleBuf, titleVars); err != nil {
		return nil, err
	}

	return titleBuf.Bytes(), nil
}

func (server *Server) readInitMessage(conn *websocket.Conn) (string, error) {
	typ, initLine, err := conn.ReadMessage()
	if err != nil {
		return "", fmt.Errorf("failed to authenticate websocket connection")
	}
	if typ != websocket.TextMessage {
		return "", fmt.Errorf("failed to authenticate websocket connection: invalid message type")
	}

	var init types.InitMessage
	if json.Unmarshal(initLine, &init) != nil {
		return "", fmt.Errorf("failed to authenticate websocket connection")
	}
	if server.options.Credential != "" && init.AuthToken != server.options.Credential {
		return "", fmt.Errorf("failed to authenticate websocket connection")
	}

	return init.Arguments, nil
}

func parseQuery(arguments string) (url.Values, error) {
	queryPath := "?"
	if arguments != "" {
		queryPath = arguments
	}
	refURL, err := url.Parse(queryPath)
	if err != nil {
		return nil, fmt.Errorf("bad arguments: %s", arguments)
	}
	return refURL.Query(), nil
}
