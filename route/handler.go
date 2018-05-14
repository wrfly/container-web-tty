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

func (server *Server) generateHandleWS(ctx context.Context,
	cancel context.CancelFunc, counter *counter, container types.Container) http.HandlerFunc {

	go func() {
		select {
		case <-counter.timer().C:
			cancel()
		case <-ctx.Done():
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		num := counter.add(1)
		closeReason := "unknown reason"

		defer func() {
			num := counter.done()
			log.Infof(
				"Connection closed by %s: %s, connections: %d/%d",
				closeReason, r.RemoteAddr, num, server.options.MaxConnection,
			)
		}()

		if int64(server.options.MaxConnection) != 0 {
			if num > server.options.MaxConnection {
				closeReason = "exceeding max number of connections"
				return
			}
		}

		log.Infof("New client connected: %s, connections: %d/%d", r.RemoteAddr, num, server.options.MaxConnection)

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

		sh := ""
		cID := container.ID

		if server.containerCli.BashExist(r.Context(), cID) {
			sh = "bash"
		} else if server.containerCli.ShExist(r.Context(), cID) {
			sh = "sh"
		}

		if sh == "" {
			log.Errorf("cannot find sh or bash in container [%s]", cID)
			return
		}

		args := []string{cID, sh}

		err = server.processWSConn(ctx, conn, container, args)

		switch err {
		case ctx.Err():
			closeReason = "cancelation"
		case webtty.ErrSlaveClosed:
			closeReason = server.factory.Name()
		case webtty.ErrMasterClosed:
			closeReason = "client"
		default:
			closeReason = fmt.Sprintf("an error: %s", err)
		}
	}
}

func (server *Server) processWSConn(ctx context.Context, conn *websocket.Conn,
	container types.Container, args []string) error {
	typ, initLine, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to authenticate websocket connection")
	}
	if typ != websocket.TextMessage {
		return fmt.Errorf("failed to authenticate websocket connection: invalid message type")
	}

	var init InitMessage
	err = json.Unmarshal(initLine, &init)
	if err != nil {
		return fmt.Errorf("failed to authenticate websocket connection")
	}
	// if init.AuthToken != server.options.Credential {
	// 	return fmt.Errorf("failed to authenticate websocket connection")
	// }

	var slave Slave
	slave, err = server.factory.New(map[string][]string{
		"arg": args,
	})
	if err != nil {
		return fmt.Errorf("failed to create backend: %s", err)
	}
	defer slave.Close()

	titleVars := server.titleVariables(
		[]string{"server", "master", "slave"},
		map[string]map[string]interface{}{
			"server": map[string]interface{}{
				"containerIP":   container.IPs[0],
				"containerName": container.Name,
				"containerID":   container.ID,
			},
			"master": map[string]interface{}{
				"remote_addr": conn.RemoteAddr(),
			},
			"slave": slave.WindowTitleVariables(),
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

	tty, err := webtty.New(&wsWrapper{conn}, slave, opts...)
	if err != nil {
		return fmt.Errorf("failed to create webtty: %s", err)
	}

	err = tty.Run(ctx)

	return err
}

func (server *Server) handleExec(c *gin.Context) {
	titleVars := server.titleVariables(
		[]string{"server"},
		map[string]map[string]interface{}{
			"server": map[string]interface{}{
				"containerName": "Name",
				"containerID":   "ID",
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
	}

	listBuf := new(bytes.Buffer)
	err := listTemplate.Execute(listBuf, listVars)
	if err != nil {
		c.Error(err)
	}

	c.Writer.Write(listBuf.Bytes())
}
