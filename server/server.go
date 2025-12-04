package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/config"
	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/nubolang/nubo/server/modules"
	"github.com/nubolang/nubo/server/router"
	"github.com/nubolang/nubo/version"
	"go.uber.org/zap"
)

const ServerPrefix = "@server/"

type Server struct {
	root  string
	isDir bool

	colorMode bool
	router    *router.Router

	cache map[string]*NodeCache
	sem   chan struct{}

	mu sync.RWMutex
}

func New(root string) (*Server, error) {
	var r *router.Router

	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	isDir := info.IsDir()

	if isDir {
		r = router.New(filepath.Clean(root))
		if err := r.Reload(); err != nil {
			return nil, err
		}
	}

	srv := &Server{
		root:      root,
		isDir:     isDir,
		colorMode: color.NoColor,
		router:    r,
		cache:     make(map[string]*NodeCache),
		sem:       make(chan struct{}, config.Current.Runtime.Server.MaxConcurrency),
	}
	zap.L().Info("server.new", zap.String("root", root), zap.Bool("isDir", isDir))
	return srv, nil
}

// Serve starts the server
func (s *Server) Serve(addr string) error {
	s.sem <- struct{}{}        // acquire
	defer func() { <-s.sem }() // release

	blue := color.New(color.FgBlue, color.Bold)
	mode := "PROD"
	if os.Getenv("NUBO_DEV") == "true" {
		mode = "DEV"
	}

	fmt.Printf("%s\n", blue.Sprint("Nubo Web - ", version.Version))
	color.New(color.FgYellow).Printf("Server listening on %s\n", addr)
	fmt.Println(color.New(color.FgHiWhite).Sprintf("Mode: %s | LogLevel: %s", mode, os.Getenv("NUBO_LOG")))
	color.New(color.FgRed).Printf("Press Ctrl+C to quit\n\n")

	zap.L().Info("server.serve.start", zap.String("addr", addr), zap.String("mode", mode))

	err := http.ListenAndServe(addr, s)
	if err != nil {
		zap.L().Error("server.serve.error", zap.String("addr", addr), zap.Error(err))
	}
	return err
}

// ServeHTTP serves the http request
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var cached bool
	start := time.Now()
	zap.L().Debug("server.request.start", zap.String("method", r.Method), zap.String("path", r.URL.Path))

	if os.Getenv("NUBO_DEV") == "true" && s.isDir {
		_ = s.router.Reload()
	}

	defer func() {
		if rcv := recover(); rcv != nil {
			stack := debug.Stack()
			zap.L().Error("server.request.panic", zap.Any("recover", rcv), zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.String("stack", string(stack)))

			w.Write(fmt.Appendf([]byte{}, "Nubo - Internal Server Error:\n%s\nStack Trace:\n%s", rcv, string(stack)))
			w.WriteHeader(http.StatusInternalServerError)

			if os.Getenv("NUBO_DEV") == "true" {
				doLog(start, r.Method, r.URL.Path, cached)
			}
		}
	}()

	// Log the request
	defer func() {
		color.NoColor = s.colorMode
		if os.Getenv("NUBO_DEV") == "true" {
			doLog(start, r.Method, r.URL.Path, cached)
		}
	}()

	defer func() {
		zap.L().Debug("server.request.finish", zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Duration("duration", time.Since(start)), zap.Bool("cached", cached))
	}()

	color.NoColor = true

	// Set the version header
	w.Header().Set("Server", "Nubo/"+version.Version)

	var file string

	if s.isDir {
		route, ok := s.router.Match(r.URL.Path)
		if !ok {
			err := serveStatic(w, r)
			if err != nil {
				zap.L().Warn("server.request.routeMissing", zap.String("path", r.URL.Path), zap.Error(err))
				s.handleError(errNotFound, w, r)
			}
			return
		}

		if !route.IsExecutable {
			http.ServeFile(w, r, route.FilePath)
			return
		}

		ctx := context.WithValue(r.Context(), "__params__", route.Params)
		r = r.WithContext(ctx)

		file = route.FilePath
	} else {
		file = s.root
	}

	nodes, c, err := s.getFile(file)
	if err != nil {
		zap.L().Error("server.request.parse", zap.String("file", file), zap.Error(err))
		s.handleError(err, w, r)
		return
	}

	cached = c
	zap.L().Debug("server.request.nodes", zap.String("file", file), zap.Bool("cached", cached))

	var eventProvider events.Provider
	if config.Current.Runtime.Events.Enabled {
		eventProvider = events.NewDefaultProvider()
	}

	run := runtime.New(eventProvider)
	zap.L().Debug("server.runtime.created", zap.Bool("events", eventProvider != nil))

	// Bind the response object to the runtime
	res := modules.NewResponse(w, r)
	run.ProvidePackage(ServerPrefix+"response", res.Pkg())
	req, err := modules.NewRequest(r)
	if err != nil {
		zap.L().Error("server.request.module", zap.String("module", "request"), zap.Error(err))
		s.handleError(err, w, r)
		return
	}

	run.ProvidePackage(ServerPrefix+"request", req)

	_, err = run.Interpret(file, nodes)
	if err != nil {
		zap.L().Error("server.runtime.interpretError", zap.String("file", file), zap.Error(err))
		s.handleError(err, w, r)
		return
	}

	// Sync and output the generated data
	res.Sync()
	zap.L().Debug("server.response.sync", zap.String("path", r.URL.Path))
}
