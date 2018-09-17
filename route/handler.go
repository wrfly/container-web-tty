package route

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/yudai/gotty/webtty"

	"github.com/wrfly/container-web-tty/types"
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
			log.Infof("Connection closed by %s: %s, connections: %d",
				closeReason, r.RemoteAddr, num,
			)
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

	if q, err := parseQuery(strings.TrimSpace(arguments)); err != nil {
		return err
	} else {
		container.ExecCMD = q.Get("cmd")
	}

	containerTTY, err := server.containerCli.Exec(ctx, container)
	if err != nil {
		return fmt.Errorf("exec container error: %s", err)
	}
	defer containerTTY.Exit()

	// handle timeout
	tout := server.options.Timeout
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
	tty, err := webtty.New(wrapper, containerTTY, opts...)
	if err != nil {
		return fmt.Errorf("failed to create webtty: %s", err)
	}

	return tty.Run(ctx)
}

func (server *Server) handleWSIndex(c *gin.Context) {
	cInfo := server.containerCli.GetInfo(c.Request.Context(), c.Param("id"))
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
		ID:     c.Param("id"),
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
		newSlave(logsReadCloser),
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
		if err != webtty.ErrMasterClosed {
			log.Errorf("failed to run webtty: %s", err)
		}
		return
	}
}

func (server *Server) makeTitleBuff(c types.Container) ([]byte, error) {
	location := "127.0.0.1"
	if c.LocServer != "" {
		location = c.LocServer
	}

	titleVars := server.titleVariables(
		[]string{"server"},
		map[string]map[string]interface{}{
			"server": map[string]interface{}{
				"containerLoc":  location,
				"containerName": c.Name,
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
