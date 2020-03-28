package route

import "github.com/wrfly/container-web-tty/util"

func (server *Server) getContainerID(execID string) (string, bool) {
	server.m.Lock()
	containerID, ok := server.execs[execID]
	server.m.Unlock()
	return containerID, ok
}

func (server *Server) setContainerID(containerID string) string {
	execID := util.ID(containerID)
	server.m.Lock()
	server.execs[execID] = containerID
	server.m.Unlock()
	return execID
}
