package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/internal/ast/astnode"
	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/server/modules"
	"go.uber.org/zap"
)

var errNotFound = errors.New("not found")

// handleError handles the error
func (s *Server) handleError(err error, w http.ResponseWriter, r *http.Request) {
	var statusCode = http.StatusInternalServerError
	fields := []zap.Field{
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.Error(err),
	}

	var exc *exception.Expection

	if errors.As(err, &exc) {
		zap.L().Error("server.request.exception", fields...)
		htmlErr := exc.HTML()

		if prefersJSON(r) {
			message, err := exc.JSON()
			if err == nil {
				w.Header().Add("Content-Type", "application/json")
				w.WriteHeader(statusCode)
			}
			_, _ = w.Write(message)
			return
		}

		w.Header().Add("Content-Type", "text/html")
		w.WriteHeader(statusCode)
		page := htmlErr.GetPage()
		_, _ = w.Write([]byte(page))
		return
	}

	if errors.Is(err, errNotFound) {
		statusCode = http.StatusNotFound
		zap.L().Warn("server.request.notFound", append(fields, zap.Int("status", statusCode))...)
	} else {
		zap.L().Error("server.request.error", append(fields, zap.Int("status", statusCode))...)
	}

	if s.isDir {
		errorFile := filepath.Join(s.root, "error.nubo")
		errNodes, _, e := s.getFile(errorFile)
		if e == nil {
			if err := s.customError(errorFile, errNodes, statusCode, err.Error(), w, r); err == nil {
				return
			} else {
				zap.L().Warn("error.nubo failed to serve error", zap.Error(err))
			}
		} else if !errors.Is(e, os.ErrNotExist) {
			zap.L().Warn("server.error.custom.load", zap.String("file", errorFile), zap.Error(e))
		}
	}

	http.Error(w, err.Error(), statusCode)
}

func (s *Server) customError(file string, nodes []*astnode.Node, status int, message string, w http.ResponseWriter, r *http.Request) error {
	zap.L().Debug("server.error.custom", zap.Int("status", status), zap.String("message", message))

	run, res, err := s.newRequestRuntime(w, r)
	if err != nil {
		return err
	}
	if err := res.SetStatus(status); err != nil {
		return err
	}

	errObj, err := modules.NewError(status, message)
	if err != nil {
		return err
	}
	run.ProvidePackage(ServerPrefix+"error", errObj)

	_, err = run.Interpret(file, nodes)
	if err != nil {
		return err
	}

	// Sync and output the generated data
	return res.Sync()
}

func prefersJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	if accept == "" {
		return false
	}

	parts := strings.Split(accept, ",")
	for _, part := range parts {
		p := strings.TrimSpace(strings.Split(part, ";")[0])
		if p == "application/json" {
			return true
		}
	}
	return false
}
