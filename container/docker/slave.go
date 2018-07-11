package docker

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	apiTypes "github.com/docker/docker/api/types"
)

// // Slave represents a PTY slave, typically it's a local command.
// type Slave interface {
// 	io.ReadWriter

// 	// WindowTitleVariables returns any values that can be used to fill out
// 	// the title of a terminal.
// 	WindowTitleVariables() map[string]interface{}

// 	// ResizeTerminal sets a new size of the terminal.
// 	ResizeTerminal(columns int, rows int) error
// }

// execInjector implement webtty.Slave
type execInjector struct {
	hResp   apiTypes.HijackedResponse
	command string
	argv    []string

	closeSignal  syscall.Signal
	closeTimeout time.Duration

	cmd       *exec.Cmd
	pty       *os.File
	ptyClosed chan struct{}
}

func newExecInjector(resp apiTypes.HijackedResponse) *execInjector {
	return &execInjector{
		hResp: resp,
	}
}

func (enj *execInjector) Read(p []byte) (n int, err error) {
	return enj.hResp.Reader.Read(p)
}

func (enj *execInjector) Write(p []byte) (n int, err error) {
	return enj.hResp.Conn.Write(p)
}

func (enj *execInjector) Close() error {
	return enj.hResp.Conn.Close()
}

func (enj *execInjector) Exit() error {
	enj.Write([]byte("exit\n"))
	return enj.hResp.Conn.Close()
}

func (enj *execInjector) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{
		"command": enj.command,
		"argv":    enj.argv,
		"pid":     enj.cmd.Process.Pid,
	}
}

func (enj *execInjector) ResizeTerminal(width int, height int) error {
	// window := struct {
	// 	row uint16
	// 	col uint16
	// 	x   uint16
	// 	y   uint16
	// }{
	// 	uint16(height),
	// 	uint16(width),
	// 	0,
	// 	0,
	// }
	// _, _, errno := syscall.Syscall(
	// 	syscall.SYS_IOCTL,
	// 	enj.hResp.Fd(),
	// 	syscall.TIOCSWINSZ,
	// 	uintptr(unsafe.Pointer(&window)),
	// )
	// if errno != 0 {
	// 	return errno
	// } else {
	// 	return nil
	// }
	return nil
}

// func runExec(dockerCli *command.DockerCli, opts *execOptions, container string, execCmd []string) error {
// 	execConfig, err := parseExec(opts, execCmd)
// 	// just in case the ParseExec does not exit
// 	if container == "" || err != nil {
// 		return cli.StatusError{StatusCode: 1}
// 	}

// 	if opts.detachKeys != "" {
// 		dockerCli.ConfigFile().DetachKeys = opts.detachKeys
// 	}

// 	// Send client escape keys
// 	execConfig.DetachKeys = dockerCli.ConfigFile().DetachKeys

// 	ctx := context.Background()
// 	client := dockerCli.Client()

// 	response, err := client.ContainerExecCreate(ctx, container, *execConfig)
// 	if err != nil {
// 		return err
// 	}

// 	execID := response.ID
// 	if execID == "" {
// 		fmt.Fprintln(dockerCli.Out(), "exec ID empty")
// 		return nil
// 	}

// 	//Temp struct for execStart so that we don't need to transfer all the execConfig
// 	if !execConfig.Detach {
// 		if err := dockerCli.In().CheckTty(execConfig.AttachStdin, execConfig.Tty); err != nil {
// 			return err
// 		}
// 	} else {
// 		execStartCheck := types.ExecStartCheck{
// 			Detach: execConfig.Detach,
// 			Tty:    execConfig.Tty,
// 		}

// 		if err := client.ContainerExecStart(ctx, execID, execStartCheck); err != nil {
// 			return err
// 		}
// 		// For now don't print this - wait for when we support exec wait()
// 		// fmt.Fprintf(dockerCli.Out(), "%s\n", execID)
// 		return nil
// 	}

// 	// Interactive exec requested.
// 	var (
// 		out, stderr io.Writer
// 		in          io.ReadCloser
// 		errCh       chan error
// 	)

// 	if execConfig.AttachStdin {
// 		in = dockerCli.In()
// 	}
// 	if execConfig.AttachStdout {
// 		out = dockerCli.Out()
// 	}
// 	if execConfig.AttachStderr {
// 		if execConfig.Tty {
// 			stderr = dockerCli.Out()
// 		} else {
// 			stderr = dockerCli.Err()
// 		}
// 	}

// 	resp, err := client.ContainerExecAttach(ctx, execID, *execConfig)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Close()
// 	errCh = promise.Go(func() error {
// 		return holdHijackedConnection(ctx, dockerCli, execConfig.Tty, in, out, stderr, resp)
// 	})

// 	if execConfig.Tty && dockerCli.In().IsTerminal() {
// 		if err := MonitorTtySize(ctx, dockerCli, execID, true); err != nil {
// 			fmt.Fprintln(dockerCli.Err(), "Error monitoring TTY size:", err)
// 		}
// 	}

// 	if err := <-errCh; err != nil {
// 		logrus.Debugf("Error hijack: %s", err)
// 		return err
// 	}

// 	var status int
// 	if _, status, err = getExecExitCode(ctx, client, execID); err != nil {
// 		return err
// 	}

// 	if status != 0 {
// 		return cli.StatusError{StatusCode: status}
// 	}

// 	return nil
// }

// // getExecExitCode perform an inspect on the exec command. It returns
// // the running state and the exit code.
// func getExecExitCode(ctx context.Context, client apiclient.ContainerAPIClient, execID string) (bool, int, error) {
// 	resp, err := client.ContainerExecInspect(ctx, execID)
// 	if err != nil {
// 		// If we can't connect, then the daemon probably died.
// 		if !apiclient.IsErrConnectionFailed(err) {
// 			return false, -1, err
// 		}
// 		return false, -1, nil
// 	}

// 	return resp.Running, resp.ExitCode, nil
// }

// // parseExec parses the specified args for the specified command and generates
// // an ExecConfig from it.
// func parseExec(opts *execOptions, execCmd []string) (*types.ExecConfig, error) {
// 	execConfig := &types.ExecConfig{
// 		User:       opts.user,
// 		Privileged: opts.privileged,
// 		Tty:        opts.tty,
// 		Cmd:        execCmd,
// 		Detach:     opts.detach,
// 	}

// 	// If -d is not set, attach to everything by default
// 	if !opts.detach {
// 		execConfig.AttachStdout = true
// 		execConfig.AttachStderr = true
// 		if opts.interactive {
// 			execConfig.AttachStdin = true
// 		}
// 	}

// 	if opts.env != nil {
// 		execConfig.Env = opts.env.GetAll()
// 	}

// 	return execConfig, nil
// }
