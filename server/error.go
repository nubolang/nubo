package server

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/events"
	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/runtime"
	"github.com/nubolang/nubo/server/modules"
	"go.uber.org/zap"
)

var errNotFound = errors.New("not found")

// handleError handles the error
func (s *Server) handleError(err error, w http.ResponseWriter, r *http.Request) {
	var statusCode = http.StatusInternalServerError

	var de *debug.DebugErr

	if errors.As(err, &de) {
		if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "application/json") {
			message, isJSON := de.JSONError()
			if isJSON {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(statusCode)
			}
			_, _ = w.Write([]byte(message))
			return
		}

		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(statusCode)
		htmlErr := de.HtmlError()
		if htmlErr == nil {
			_, _ = w.Write([]byte(de.Error()))
			return
		}
		_, _ = w.Write([]byte(htmlErr.Error()))
		return
	}

	if errors.Is(err, errNotFound) {
		statusCode = http.StatusNotFound
	}

	if s.isDir {
		errNodes, _, e := s.getFile(filepath.Join(s.root, "error.nubo"))
		if e == nil {
			if err := s.customError(errNodes, statusCode, err.Error(), w, r); err == nil {
				return
			} else {
				zap.L().Warn("error.nubo failed to serve error", zap.Error(err))
			}
		}
	}

	http.Error(w, err.Error(), statusCode)
}

func (s *Server) customError(nodes []*astnode.Node, status int, message string, w http.ResponseWriter, r *http.Request) error {
	run := runtime.New(events.NewDefaultProvider())

	// Bind the response object to the runtime
	res := modules.NewResponse(w, r)
	run.ProvidePackage(ServerPrefix+"response", res.Pkg())
	req, err := modules.NewRequest(r)
	if err != nil {
		return err
	}

	run.ProvidePackage(ServerPrefix+"request", req)
	run.ProvidePackage(ServerPrefix+"error", modules.NewError(status, message))

	_, err = run.Interpret("error.nubo", nodes)
	if err != nil {
		return err
	}

	// Sync and output the generated data
	res.Sync()
	return nil
}
