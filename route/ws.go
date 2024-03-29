package route

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (server *Server) handleWSIndex(c *gin.Context) {
	var (
		containerID  = c.Param("cid")
		execID       = c.Param("eid")
		_foundExecID bool
	)
	if containerID == "" {
		containerID, _foundExecID = server.getContainerID(execID)
		if !_foundExecID {
			c.String(http.StatusBadRequest, fmt.Sprintf("exec id %s not found", execID))
			return
		}
	}
	cInfo := server.containerCli.GetInfo(c.Request.Context(), containerID)
	titleVars := server.titleVariables(
		[]string{"server"},
		map[string]map[string]interface{}{
			"server": {
				"containerName": cInfo.Name,
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
		"base":  strings.TrimSuffix(server.options.Base, "/"),
	}

	indexBuf := new(bytes.Buffer)
	err = indexTemplate.Execute(indexBuf, indexVars)
	if err != nil {
		c.Error(err)
	}

	c.Writer.Write(indexBuf.Bytes())
}
