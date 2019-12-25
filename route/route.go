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
	"sync"
	noesctmpl "text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/yudai/gotty/webtty"

	"github.com/wrfly/container-web-tty/config"
	"github.com/wrfly/container-web-tty/container"
	"github.com/wrfly/container-web-tty/route/asset"
	"github.com/wrfly/container-web-tty/types"
)

// Server provides a webtty HTTP endpoint.
type Server struct {
	options      config.ServerConfig
	containerCli container.Cli
	upgrader     *websocket.Upgrader
	srv          *http.Server
	hostname     string

	masters map[string]*types.ShareTTY
	mMux    sync.RWMutex
}

var (
	indexTemplate *template.Template
	listTemplate  *template.Template
	titleTemplate *noesctmpl.Template
)

func mod(i, j int) int {
	return i % j
}

func init() {
	indexData, err := asset.Find("/index.html")
	if err != nil {
		log.Fatal(err)
	}
	indexTemplate = indexData.Template()

	listIndexData, err := asset.Find("/list.html")
	if err != nil {
		log.Fatal(err)
	}

	listTemplate, err = template.New(listIndexData.Name()).
		Funcs(template.FuncMap{
			"mod": mod,
		}).Parse(string(listIndexData.Bytes()))
	if err != nil {
		panic(err)
	}

	titleFormat := "{{ .containerName }} - {{ printf \"%.8s\" .containerID }}@{{ .containerLoc }}"
	titleTemplate, err = noesctmpl.New("title").Parse(titleFormat)
	if err != nil {
		log.Fatal(err)
	}
}

// New creates a new instance of Server.
// Server will use the New() of the factory provided to handle each request.
func New(containerCli container.Cli, options config.ServerConfig) (*Server, error) {

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
		masters:      make(map[string]*types.ShareTTY, 50),
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

	router := gin.New()
	router.Use(gin.Recovery())
	if gin.Mode() == gin.DebugMode {
		router.Use(gin.Logger())
	}

	// Routes
	router.GET("/", server.handleListContainers)
	router.GET("/auth_token.js", server.handleAuthToken)
	router.GET("/config.js", server.handleConfig)

	h := gin.WrapF(asset.Handler)
	for _, f := range asset.List() {
		if f.Name() != "/" {
			router.GET(f.Name(), h)
		}
	}

	// exec
	counter := newCounter(server.options.IdleTime)
	router.GET("/exec/:id/", server.terminalPage)
	router.GET("/exec/:id/"+"ws", func(c *gin.Context) { server.handleExec(c, counter) })

	if server.options.EnableShare {
		// share screen
		router.GET("/share/:id/", server.terminalPage)
		router.GET("/share/:id/ws", func(c *gin.Context) { server.handleShare(c) })
	}

	// logs
	router.GET("/logs/:id/", server.terminalPage)
	router.GET("/logs/:id/"+"ws", func(c *gin.Context) { server.handleLogs(c) })

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

	hostPort := net.JoinHostPort(server.options.Address,
		fmt.Sprint(server.options.Port))
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
