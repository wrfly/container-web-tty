package route

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/types"
)

const listedID = "1111111111111111111111111111111111111111111111111111111111111111"

// fakeCli only knows the containers of its list
type fakeCli struct {
	cs      *types.Containers
	started []string
}

func newFakeCli() *fakeCli {
	cs := &types.Containers{}
	cs.Set([]types.Container{{ID: listedID, Name: "listed", Shell: "/bin/sh"}})
	return &fakeCli{cs: cs}
}

func (f *fakeCli) GetInfo(ctx context.Context, cid string) types.Container { return f.cs.Find(cid) }
func (f *fakeCli) List(context.Context) []types.Container                  { return f.cs.List() }
func (f *fakeCli) Start(ctx context.Context, cid string) error {
	f.started = append(f.started, cid)
	return nil
}
func (f *fakeCli) Stop(ctx context.Context, cid string) error    { return nil }
func (f *fakeCli) Restart(ctx context.Context, cid string) error { return nil }
func (f *fakeCli) Exec(ctx context.Context, c types.Container) (types.TTY, error) {
	return nil, io.EOF
}
func (f *fakeCli) Close() error { return nil }
func (f *fakeCli) Logs(ctx context.Context, opts types.LogOptions) (io.ReadCloser, error) {
	return nil, io.EOF
}

func TestOnlyListedContainers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cli := newFakeCli()
	server, err := New(cli, config.ServerConfig{Base: "/"})
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/e/:cid/", server.handleExecRedirect)
	router.GET("/logs/:cid/", server.handleWSIndex)
	router.POST("/container/start/:id", server.handleStartContainer)

	do := func(method, path string) int {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
		return w.Code
	}

	for _, cid := range []string{listedID, listedID[:12], "listed"} {
		if code := do("GET", "/e/"+cid+"/"); code != http.StatusFound {
			t.Errorf("exec %q: want 302, got %d", cid, code)
		}
		if code := do("GET", "/logs/"+cid+"/"); code != http.StatusOK {
			t.Errorf("logs %q: want 200, got %d", cid, code)
		}
	}
	for id := range server.execs {
		if server.execs[id] != listedID {
			t.Errorf("exec %s should map to the full container ID, got %s", id, server.execs[id])
		}
	}

	// short prefix and containers out of the list
	for _, cid := range []string{"1", "11111", "222222222222", "unlisted"} {
		if code := do("GET", "/e/"+cid+"/"); code != http.StatusNotFound {
			t.Errorf("exec %q: want 404, got %d", cid, code)
		}
		if code := do("GET", "/logs/"+cid+"/"); code != http.StatusNotFound {
			t.Errorf("logs %q: want 404, got %d", cid, code)
		}
		if code := do("POST", "/container/start/"+cid); code != http.StatusNotFound {
			t.Errorf("start %q: want 404, got %d", cid, code)
		}
	}
	if len(cli.started) != 0 {
		t.Errorf("unexpected container start: %v", cli.started)
	}
}

func TestExecOptions(t *testing.T) {
	q := url.Values{
		"cmd":  {"ls"},
		"env":  {"A=1"},
		"user": {"root"},
		"p":    {"1"},
	}

	server := &Server{}
	opts := server.execOptions(q)
	if opts.Cmd != "ls" || opts.Env != "A=1" {
		t.Errorf("cmd and env should be kept: %+v", opts)
	}
	if opts.User != "" || opts.Privileged {
		t.Errorf("user and privileged should be ignored by default: %+v", opts)
	}

	server.options.ExecPrivilege = true
	opts = server.execOptions(q)
	if opts.User != "root" || !opts.Privileged {
		t.Errorf("user and privileged should be accepted when enabled: %+v", opts)
	}
}
