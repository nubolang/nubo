package server

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/nubolang/nubo/config"
	"github.com/nubolang/nubo/internal/exception"
	"go.uber.org/zap"
)

func errorTestServer(t *testing.T, source string, dev bool) (*Server, *bytes.Buffer) {
	t.Helper()
	t.Setenv("NUBO_DEV", strconv.FormatBool(dev))

	previousConfig := config.Current
	settings := &config.Config{}
	settings.ApplyDefaults()
	settings.Runtime.Events.Enabled = false
	config.Current = settings
	t.Cleanup(func() { config.Current = previousConfig })

	restoreLogger := zap.ReplaceGlobals(zap.NewNop())
	t.Cleanup(restoreLogger)

	output := &bytes.Buffer{}
	previousOutput := log.Writer()
	log.SetOutput(output)
	t.Cleanup(func() { log.SetOutput(previousOutput) })

	root := t.TempDir()
	if source != "" {
		if err := os.WriteFile(filepath.Join(root, "error.nubo"), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}

	srv, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	return srv, output
}

func TestCustomErrorPageHandlesRequestErrors(t *testing.T) {
	for _, dev := range []bool{false, true} {
		for _, tc := range []struct {
			name   string
			err    error
			status int
		}{
			{"missing", fmt.Errorf("wrapped: %w", errNotFound), 404},
			{"internal", errors.New("private request detail"), 500},
			{"exception", exception.Create("private request detail"), 500},
			{"status", exception.Create("invalid input").WithStatusCode(422), 422},
		} {
			t.Run(fmt.Sprintf("%s/dev=%t", tc.name, dev), func(t *testing.T) {
				srv, _ := errorTestServer(t, `
import response from "@server/response"
import error from "@server/error"
response.write("custom:" + string(error.status) + ":" + error.message)
`, dev)
				response := httptest.NewRecorder()
				srv.handleError(tc.err, response, httptest.NewRequest("GET", "/failed", nil))

				if response.Code != tc.status || !strings.HasPrefix(response.Body.String(), fmt.Sprintf("custom:%d:", tc.status)) {
					t.Fatalf("custom page was skipped: %d %s", response.Code, response.Body.String())
				}

				if !dev && strings.Contains(response.Body.String(), "private request detail") {
					t.Fatal("production response exposed internal details")
				}
			})
		}
	}
}

func TestCustomErrorFailureHasDiagnostics(t *testing.T) {
	for _, dev := range []bool{false, true} {
		for _, source := range []string{
			`panic("error page broke")`,
			"fn unfinished() int",
			"// no response body\n",
			`import response from "@server/response"
response.header("X-Partial", "discard")
response.redirect("/discard")
panic("error page broke")`,
		} {
			t.Run(fmt.Sprintf("dev=%t/%s", dev, source), func(t *testing.T) {
				srv, logs := errorTestServer(t, source, dev)
				request := httptest.NewRequest("GET", "/failed", nil)
				request.Header.Set("Accept", "application/json")
				response := httptest.NewRecorder()
				srv.handleError(errors.New("original request broke"), response, request)

				if response.Code != 500 || response.Header().Get("Location") != "" || response.Header().Get("X-Partial") != "" {
					t.Fatalf("failed page committed a partial response: %d %v", response.Code, response.Header())
				}

				for _, detail := range []string{"original request broke", "server.error.custom.failed", "error.nubo"} {
					if !strings.Contains(logs.String(), detail) {
						t.Fatalf("missing %q in logs: %s", detail, logs.String())
					}
				}

				body := response.Body.String()
				if dev {
					if !strings.Contains(body, "original request broke") || !strings.Contains(body, "Error page failed") {
						t.Fatalf("development response hid the failures: %s", body)
					}
				} else if strings.Contains(body, "original request broke") || !strings.Contains(body, "Internal Server Error") {
					t.Fatalf("unexpected production fallback: %s", body)
				}
			})
		}
	}
}

func TestMissingErrorPageUsesFallback(t *testing.T) {
	srv, logs := errorTestServer(t, "", false)
	response := httptest.NewRecorder()
	srv.handleError(errors.New("original failure"), response, httptest.NewRequest("GET", "/failed", nil))

	if response.Code != 500 || response.Body.String() != "Internal Server Error" {
		t.Fatalf("unexpected fallback: %d %s", response.Code, response.Body.String())
	}

	if !strings.Contains(logs.String(), "original failure") || strings.Contains(logs.String(), "server.error.custom.failed") {
		t.Fatalf("missing page was treated as a failed custom page: %s", logs.String())
	}
}

func TestErrorPagePanicUsesFallback(t *testing.T) {
	srv, logs := errorTestServer(t, `panic("page error")`, false)
	srv.cache = nil
	response := httptest.NewRecorder()
	srv.handleError(errors.New("original failure"), response, httptest.NewRequest("GET", "/failed", nil))

	if response.Code != 500 || !strings.Contains(logs.String(), "error page panicked") {
		t.Fatalf("error page panic was not handled: %d %s", response.Code, logs.String())
	}
}

func TestRequestErrorsUseCustomPage(t *testing.T) {
	for _, source := range []string{"panic(\"route failed\")\n", "fn unfinished() int"} {
		t.Run(source, func(t *testing.T) {
			srv, _ := errorTestServer(t, `import response from "@server/response"
response.write("custom request failure")`, false)
			if err := os.WriteFile(filepath.Join(srv.root, "index.nubo"), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			if err := srv.router.Reload(); err != nil {
				t.Fatal(err)
			}

			response := httptest.NewRecorder()
			srv.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))

			if response.Code != 500 || response.Body.String() != "custom request failure" {
				t.Fatalf("request skipped the custom page: %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestRequestPanicUsesCustomPage(t *testing.T) {
	srv, logs := errorTestServer(t, `import response from "@server/response"
response.write("custom panic failure")`, false)
	srv.router = nil

	response := httptest.NewRecorder()
	srv.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))

	if response.Code != 500 || response.Body.String() != "custom panic failure" || !strings.Contains(logs.String(), "panic:") {
		t.Fatalf("request panic skipped the custom page: %d %s", response.Code, response.Body.String())
	}
}

func TestSingleFileServerUsesSiblingErrorPage(t *testing.T) {
	srv, _ := errorTestServer(t, `import response from "@server/response"
response.write("sibling error page")`, false)
	srv.root = filepath.Join(srv.root, "index.nubo")
	srv.isDir = false

	response := httptest.NewRecorder()
	srv.handleError(errors.New("route failed"), response, httptest.NewRequest("GET", "/", nil))

	if response.Code != 500 || response.Body.String() != "sibling error page" {
		t.Fatalf("single-file server skipped the error page: %d %s", response.Code, response.Body.String())
	}
}

var _ http.ResponseWriter = (*errorResponse)(nil)
