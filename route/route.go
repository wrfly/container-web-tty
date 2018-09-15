package route

import (
	"context"
	"fmt"
	"html/template"
	"net"
	"net/http"
	pprof "net/http/pprof"
	"os"
	"regexp"
	noesctmpl "text/template"
	"time"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/yudai/gotty/webtty"

	"github.com/wrfly/container-web-tty/container"
)

// Server provides a webtty HTTP endpoint.
type Server struct {
	options      *Options
	containerCli container.Cli
	upgrader     *websocket.Upgrader
	srv          *http.Server
	hostname     string
}

var (
	indexTemplate *template.Template
	listTemplate  *template.Template
	titleTemplate *noesctmpl.Template
)

func init() {
	indexData, err := Asset("index.html")
	if err != nil {
		log.Fatal(err)
	}
	indexTemplate, err = template.New("index").Parse(string(indexData))
	if err != nil {
		log.Fatal(err)
	}

	listIndexData, err := Asset("list.html")
	if err != nil {
		log.Fatal(err)
	}
	listTemplate, err = template.New("list").Parse(string(listIndexData))
	if err != nil {
		log.Fatal(err)
	}

	titleFormat := "{{ .containerName }} - {{ printf \"%.8s\" .containerID }}@{{ .containerLoc }}"
	titleTemplate, err = noesctmpl.New("title").Parse(titleFormat)
	if err != nil {
		log.Fatal(err)
	}
}

// New creates a new instance of Server.
// Server will use the New() of the factory provided to handle each request.
func New(containerCli container.Cli, options *Options) (*Server, error) {

	var originChekcer func(r *http.Request) bool
	if options.WSOrigin != "" {
		matcher, err := regexp.Compile(options.WSOrigin)
		if err != nil {
			return nil, fmt.Errorf("failed to compile regular expression of Websocket Origin: %s", options.WSOrigin)
		}
		originChekcer = func(r *http.Request) bool {
			return matcher.MatchString(r.Header.Get("Origin"))
		}
	}

	h, _ := os.Hostname()
	return &Server{
		options:      options,
		containerCli: containerCli,
		hostname:     h,

		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    webtty.Protocols,
			CheckOrigin:     originChekcer,
		},
	}, nil
}

// Run starts the main process of the Server.
// The cancelation of ctx will shutdown the server immediately with aborting
// existing connections. Use WithGracefulContext() to support graceful shutdown.
func (server *Server) Run(ctx context.Context, options ...RunOption) error {
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	opts := &RunOptions{gracefulCtx: context.Background()}
	for _, opt := range options {
		opt(opts)
	}

	counter := newCounter(server.options.Timeout)

	router := gin.New()
	router.Use(gin.Recovery())
	if gin.Mode() == gin.DebugMode {
		router.Use(gin.Logger())
	}

	h := http.FileServer(
		&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "/"},
	)
	fh := gin.WrapH(http.StripPrefix("/", h))

	// Routes
	router.GET("/", server.handleListContainers)
	router.GET("/auth_token.js", server.handleAuthToken)
	router.GET("/config.js", server.handleConfig)

	for _, fileName := range AssetNames() {
		router.GET(fileName, fh)
	}

	// exec
	router.GET("/exec/:id/", func(c *gin.Context) { server.handleWSIndex(c) })
	router.GET("/exec/:id/"+"ws", func(c *gin.Context) {
		containerInfo := server.containerCli.GetInfo(c.Request.Context(), c.Param("id"))
		server.generateHandleWS(cctx, counter, containerInfo).ServeHTTP(c.Writer, c.Request)
	})
	// logs
	router.GET("/logs/:id/", func(c *gin.Context) {
		server.handleWSIndex(c)
	})
	router.GET("/logs/:id/"+"ws", func(c *gin.Context) {
		server.handleLogs(c)
	})

	ctl := server.options.Control
	if ctl.Enable {
		// container actions: start|stop|restart
		containerG := router.Group("/container")
		if ctl.Start || ctl.All {
			containerG.POST("/start/:id", server.handleStartContainer)
		}
		if ctl.Stop || ctl.All {
			containerG.POST("/stop/:id", server.handleStopContainer)
		}
		if ctl.Restart || ctl.All {
			containerG.POST("/restart/:id", server.handleRestartContainer)
		}
	}

	// pprof
	rootMux := http.NewServeMux()
	if log.GetLevel() == log.DebugLevel {
		rootMux.HandleFunc("/debug/pprof/", pprof.Index)
		rootMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		rootMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		rootMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		rootMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	rootMux.Handle("/", router)

	hostPort := net.JoinHostPort(server.options.Address, server.options.Port)
	srv := &http.Server{
		Addr:    hostPort,
		Handler: rootMux,
	}

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	go func() {
		select {
		case <-opts.gracefulCtx.Done():
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := srv.Shutdown(ctx); err != nil {
				log.Fatal("Server Shutdown:", err)
			}
		case <-cctx.Done():
		}
	}()

	log.Infof("Server running at http://%s", hostPort)

	var err error
	select {
	case err = <-srvErr:
		if err == http.ErrServerClosed { // by graceful ctx
			err = nil
		} else {
			cancel()
		}
	case <-cctx.Done():
		srv.Close()
		err = cctx.Err()
	}

	conn := counter.count()
	if conn > 0 {
		log.Printf("Waiting for %d connections to be closed", conn)
		fmt.Println("Ctl-C to force close")
	}
	counter.wait()

	return err
}
