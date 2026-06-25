package server

import (
	"errors"
	"fmt"
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
var errInternalServerMessage = "Internal Server Error"

// handleError handles the error
func (s *Server) handleError(err error, w http.ResponseWriter, r *http.Request) {
	devMode := os.Getenv("NUBO_DEV") == "true"
	var statusCode = http.StatusInternalServerError
	fields := []zap.Field{
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.Error(err),
	}

	var exc *exception.Expection

	if errors.As(err, &exc) {
		zap.L().Error("server.request.exception", fields...)
		if devMode {
			s.writeDevError(exc, statusCode, w, r)
		} else {
			s.writeProdError(statusCode, w, r)
		}
		return
	}

	if errors.Is(err, errNotFound) {
		statusCode = http.StatusNotFound
		zap.L().Warn("server.request.notFound", append(fields, zap.Int("status", statusCode))...)
	} else {
		zap.L().Error("server.request.error", append(fields, zap.Int("status", statusCode))...)

		if devMode {
			s.writeDevError(exception.Create("%s", err.Error()).WithBase(err).WithStatusCode(statusCode), statusCode, w, r)
		} else {
			s.writeProdError(statusCode, w, r)
		}
		return
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

func (s *Server) writeDevError(exc *exception.Expection, statusCode int, w http.ResponseWriter, r *http.Request) {
	if prefersJSON(r) {
		message, err := exc.JSON()
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			_, _ = w.Write(message)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(statusCode)
	page := exc.HTML().GetPage()
	_, _ = w.Write([]byte(page))
}

func (s *Server) writeProdError(statusCode int, w http.ResponseWriter, r *http.Request) {
	message := errInternalServerMessage
	if statusCode != http.StatusInternalServerError {
		message = http.StatusText(statusCode)
	}

	if prefersJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		fmt.Fprintf(w, "{\"status\":%d,\"message\":%q}", statusCode, message)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(message))
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
