package server

import (
	"bytes"
	"net/http"
)

// Commit the error page only after it succeeds.
type errorResponse struct {
	headers http.Header
	status  int
	body    bytes.Buffer
}

func newErrorResponse() *errorResponse {
	return &errorResponse{headers: make(http.Header)}
}

func (w *errorResponse) Header() http.Header {
	return w.headers
}

func (w *errorResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *errorResponse) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.body.Write(data)
}

func (w *errorResponse) sync(target http.ResponseWriter, r *http.Request) error {
	for name, values := range w.headers {
		target.Header()[name] = append([]string(nil), values...)
	}

	target.WriteHeader(w.status)
	if r.Method == http.MethodHead {
		return nil
	}

	_, err := target.Write(w.body.Bytes())
	return err
}
