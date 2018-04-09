package route

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/wrfly/container-web-tty/gotty/webtty"
)

func (server *Server) generateHandleWS(ctx context.Context,
	cancel context.CancelFunc, counter *counter, containerID string) http.HandlerFunc {
	once := new(int64)

	go func() {
		select {
		case <-counter.timer().C:
			cancel()
		case <-ctx.Done():
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		if server.options.Once {
			success := atomic.CompareAndSwapInt64(once, 0, 1)
			if !success {
				http.Error(w, "Server is shutting down", http.StatusServiceUnavailable)
				return
			}
		}

		num := counter.add(1)
		closeReason := "unknown reason"

		defer func() {
			num := counter.done()
			log.Printf(
				"Connection closed by %s: %s, connections: %d/%d",
				closeReason, r.RemoteAddr, num, server.options.MaxConnection,
			)

			if server.options.Once {
				cancel()
			}
		}()

		if int64(server.options.MaxConnection) != 0 {
			if num > server.options.MaxConnection {
				closeReason = "exceeding max number of connections"
				return
			}
		}

		log.Printf("New client connected: %s, connections: %d/%d", r.RemoteAddr, num, server.options.MaxConnection)

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

		sh := "sh"
		if server.containerCli.BashExist(r.Context(), containerID) {
			sh = "bash"
		}
		args := []string{containerID, sh}

		err = server.processWSConn(ctx, conn, args)

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

func (server *Server) processWSConn(ctx context.Context, conn *websocket.Conn, args []string) error {
	typ, initLine, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("failed to authenticate websocket connection")
	}
	if typ != websocket.TextMessage {
		return errors.New("failed to authenticate websocket connection: invalid message type")
	}

	var init InitMessage
	err = json.Unmarshal(initLine, &init)
	if err != nil {
		return fmt.Errorf("failed to authenticate websocket connection")
	}
	if init.AuthToken != server.options.Credential {
		return errors.New("failed to authenticate websocket connection")
	}

	var slave Slave
	slave, err = server.factory.New(map[string][]string{
		"arg": args,
	})
	if err != nil {
		return fmt.Errorf("failed to create backend")
	}
	defer slave.Close()

	titleVars := server.titleVariables(
		[]string{"server", "master", "slave"},
		map[string]map[string]interface{}{
			"server": map[string]interface{}{
				"hostname":      server.options.TitleVariables["hostname"].(string),
				"containerName": args[0],
				"containerID":   args[0],
			},
			"master": map[string]interface{}{
				"remote_addr": conn.RemoteAddr(),
			},
			"slave": slave.WindowTitleVariables(),
		},
	)

	titleBuf := new(bytes.Buffer)
	err = server.titleTemplate.Execute(titleBuf, titleVars)
	if err != nil {
		return fmt.Errorf("failed to fill window title template")
	}

	opts := []webtty.Option{
		webtty.WithWindowTitle(titleBuf.Bytes()),
	}
	if server.options.PermitWrite {
		opts = append(opts, webtty.WithPermitWrite())
	}
	// if server.options.EnableReconnect {
	// 	opts = append(opts, webtty.WithReconnect(server.options.ReconnectTime))
	// }
	// if server.options.Width > 0 {
	// 	opts = append(opts, webtty.WithFixedColumns(server.options.Width))
	// }
	// if server.options.Height > 0 {
	// 	opts = append(opts, webtty.WithFixedRows(server.options.Height))
	// }
	// if server.options.Preferences != nil {
	// 	opts = append(opts, webtty.WithMasterPreferences(server.options.Preferences))
	// }

	tty, err := webtty.New(&wsWrapper{conn}, slave, opts...)
	if err != nil {
		return fmt.Errorf("failed to create webtty")
	}

	err = tty.Run(ctx)

	return err
}

func (server *Server) handleIndex(c *gin.Context) {
	titleVars := server.titleVariables(
		[]string{"server", "master"},
		map[string]map[string]interface{}{
			"master": map[string]interface{}{
				"remote_addr": c.Request.RemoteAddr,
			},
			"server": map[string]interface{}{
				"hostname":      server.options.TitleVariables["hostname"].(string),
				"containerName": "name",
				"containerID":   "ID",
			},
		},
	)

	titleBuf := new(bytes.Buffer)
	err := server.titleTemplate.Execute(titleBuf, titleVars)
	if err != nil {
		c.Error(err)
	}

	indexVars := map[string]interface{}{
		"title": titleBuf.String(),
	}

	indexBuf := new(bytes.Buffer)
	err = server.indexTemplate.Execute(indexBuf, indexVars)
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
	titleVars := server.titleVariables(
		[]string{"server", "master"},
		map[string]map[string]interface{}{
			"master": map[string]interface{}{
				"remote_addr": c.Request.RemoteAddr,
			},
			"server": map[string]interface{}{
				"containerName": "List",
				"containerID":   "Containers",
				"hostname":      server.options.TitleVariables["hostname"].(string),
			},
		},
	)

	titleBuf := new(bytes.Buffer)
	err := server.titleTemplate.Execute(titleBuf, titleVars)
	if err != nil {
		c.Error(err)
	}

	listVars := map[string]interface{}{
		"title":      titleBuf.String(),
		"containers": server.containerCli.List(c.Request.Context()),
	}

	listBuf := new(bytes.Buffer)
	err = server.listTemplate.Execute(listBuf, listVars)
	if err != nil {
		c.Error(err)
	}

	c.Writer.Write(listBuf.Bytes())
}
