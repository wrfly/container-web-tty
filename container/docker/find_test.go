package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/moby/moby/client"

	"github.com/wrfly/container-web-tty/types"
)

// containers out of the filtered list must not be accessible,
// even if they exist on the docker host
func TestOnlyListedContainers(t *testing.T) {
	const (
		listedID   = "1111111111111111111111111111111111111111111111111111111111111111"
		unlistedID = "2222222222222222222222222222222222222222222222222222222222222222"
	)

	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/containers/json"):
			json.NewEncoder(w).Encode([]container.Summary{
				{ID: listedID, Names: []string{"/listed"}, State: "running"},
			})
		case strings.HasSuffix(r.URL.Path, "/archive"):
			w.Header().Set("X-Docker-Container-Path-Stat", "e30=") // {}
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{}"))
		}
	}))
	defer srv.Close()

	cli, err := client.NewClientWithOpts(
		client.WithHost("tcp://"+strings.TrimPrefix(srv.URL, "http://")),
		client.WithVersion("1.41"),
	)
	if err != nil {
		t.Fatal(err)
	}
	d := &DockerCli{cli: cli, containers: &types.Containers{}}
	ctx := context.Background()

	for _, cid := range []string{listedID, listedID[:12], "listed"} {
		if c := d.GetInfo(ctx, cid); c.ID != listedID {
			t.Errorf("get info %q: want %s, got %q", cid, listedID, c.ID)
		}
	}

	for _, cid := range []string{unlistedID, unlistedID[:12], "1", "11111", "unlisted"} {
		if c := d.GetInfo(ctx, cid); c.ID != "" {
			t.Errorf("get info %q: want nothing, got %s", cid, c.ID)
		}
		if err := d.Start(ctx, cid); err == nil {
			t.Errorf("start %q: want error", cid)
		}
		if _, err := d.Logs(ctx, types.LogOptions{ID: cid}); err == nil {
			t.Errorf("logs %q: want error", cid)
		}
		if _, err := d.Exec(ctx, types.Container{ID: cid, Shell: "/bin/sh"}); err == nil {
			t.Errorf("exec %q: want error", cid)
		}
	}

	for _, req := range requests {
		if strings.Contains(req, unlistedID) || strings.Contains(req, "/unlisted") ||
			(strings.Contains(req, "/containers/1") && !strings.Contains(req, listedID)) {
			t.Errorf("unexpected request to docker: %s", req)
		}
	}
}
