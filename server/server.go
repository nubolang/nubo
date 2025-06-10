package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/nubolang/nubo/server/modules"
	"github.com/nubolang/nubo/server/router"
	"github.com/nubolang/nubo/version"
)

const ServerPrefix = "@server/"

type Server struct {
	root  string
	isDir bool

	colorMode bool
	router    *router.Router

	cache map[string]*NodeCache

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

	return &Server{
		root:      root,
		isDir:     isDir,
		colorMode: color.NoColor,
		router:    r,
		cache:     make(map[string]*NodeCache),
	}, nil
}

// Serve starts the server
func (s *Server) Serve(addr string) error {
	blue := color.New(color.FgBlue, color.Bold)

	fmt.Printf("%s\n", blue.Sprint("Nubo Web - ", version.Version))
	color.New(color.FgYellow).Printf("Server listening on %s\n", addr)
	color.New(color.FgRed).Printf("Press Ctrl+C to quit\n\n")

	return http.ListenAndServe(addr, s)
}

// ServeHTTP serves the http request
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var cached bool
	start := time.Now()

	defer func() {
		if rcv := recover(); rcv != nil {
			log.Printf("PANIC RECOVERED: %v. Request: %s %s", rcv, r.Method, r.URL.Path)

			w.Write([]byte("Nubo - Internal Server Error #(PANIC)"))
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

	color.NoColor = true

	// Set the version header
	w.Header().Set("Server", "Nubo/"+version.Version)

	var file string

	if s.isDir {
		route, ok := s.router.Match(r.URL.Path)
		if !ok {
			s.handleError(errNotFound, w, r)
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
		s.handleError(err, w, r)
		return
	}

	cached = c

	run := runtime.New(events.NewDefaultProvider())

	// Bind the response object to the runtime
	res := modules.NewResponse(w)
	run.ProvidePackage(ServerPrefix+"response", res.Pkg())
	req, err := modules.NewRequest(r)
	if err != nil {
		s.handleError(err, w, r)
		return
	}

	run.ProvidePackage(ServerPrefix+"request", req)

	_, err = run.Interpret(file, nodes)
	if err != nil {
		s.handleError(err, w, r)
		return
	}

	// Sync and output the generated data
	res.Sync()
}
