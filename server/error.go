package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/nubolang/nubo/internal/exception"
	"github.com/nubolang/nubo/server/modules"
	"go.uber.org/zap"
)

var errNotFound = errors.New("not found")
var errErrorPageMissing = errors.New("error page not found")
var errInternalServerMessage = "Internal Server Error"

func (s *Server) handleError(err error, w http.ResponseWriter, r *http.Request) {
	devMode := os.Getenv("NUBO_DEV") == "true"
	statusCode := http.StatusInternalServerError
	var exc *exception.Expection

	if errors.Is(err, errNotFound) {
		statusCode = http.StatusNotFound
	} else if errors.As(err, &exc) && exc.StatusCode() >= 400 && exc.StatusCode() <= 599 {
		statusCode = exc.StatusCode()
	}

	if statusCode >= 500 {
		logServerError("server.request.error", err, r, statusCode)
	} else {
		zap.L().Warn("server.request.error", zap.Int("status", statusCode), zap.String("path", r.URL.Path), zap.Error(err))
	}

	message := http.StatusText(statusCode)
	if devMode {
		message = err.Error()
		if data, ok := exception.Unwrap(err); ok {
			message = data.Message
		}
	}

	fallback := err
	errorRoot := s.root
	if !s.isDir {
		errorRoot = filepath.Dir(s.root)
	}

	errorFile := filepath.Join(errorRoot, "error.nubo")
	output, customErr := s.customError(errorFile, statusCode, message, r)
	if customErr == nil {
		if err := output.sync(w, r); err != nil {
			logServerError("server.error.custom.write", err, r, statusCode)
		}

		return
	}

	if !errors.Is(customErr, errErrorPageMissing) {
		combined := fmt.Errorf("request error: %w\nerror page %s failed: %w", err, errorFile, customErr)
		logServerError("server.error.custom.failed", combined, r, statusCode)
		fallback = exception.Create("Request error:\n%s\n\nError page failed:\n%s", err.Error(), customErr.Error()).WithBase(combined).WithStatusCode(statusCode)
	}

	if devMode {
		s.writeDevError(exception.From(fallback, nil).WithStatusCode(statusCode), statusCode, w, r)
	} else {
		s.writeProdError(statusCode, w, r)
	}
}

func logServerError(message string, err error, r *http.Request, status int) {
	if entry := zap.L().Check(zap.ErrorLevel, message); entry != nil {
		entry.Write(zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Int("status", status), zap.Error(err))
		return
	}

	log.Printf("%s: %s %s (status %d): %v", message, r.Method, r.URL.Path, status, err)
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

func (s *Server) customError(file string, status int, message string, r *http.Request) (output *errorResponse, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			output = nil
			err = fmt.Errorf("error page panicked: %v\n%s", recovered, debug.Stack())
		}
	}()

	zap.L().Debug("server.error.custom", zap.Int("status", status), zap.String("message", message))

	nodes, _, err := s.getFile(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", errErrorPageMissing, file)
		}

		return nil, fmt.Errorf("load error page: %w", err)
	}

	output = newErrorResponse()
	run, res, err := s.newRequestRuntime(output, r)
	if err != nil {
		return nil, err
	}
	if err := res.SetStatus(status); err != nil {
		return nil, err
	}

	errObj, err := modules.NewError(status, message)
	if err != nil {
		return nil, err
	}
	run.ProvidePackage(ServerPrefix+"error", errObj)

	_, err = run.Interpret(file, nodes)
	if err != nil {
		return nil, err
	}

	if err := res.Sync(); err != nil {
		return nil, err
	}

	if output.body.Len() == 0 && output.headers.Get("Location") == "" && r.Method != http.MethodHead && output.status != http.StatusNoContent && output.status != http.StatusNotModified {
		return nil, fmt.Errorf("error page produced an empty response")
	}

	return output, nil
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
