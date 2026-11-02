package router

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testRouter(t *testing.T) *Router {
	t.Helper()
	root := filepath.Join(t.TempDir(), "routes")

	for _, name := range []string{"index.nubo", "users/new.nubo", "users/[id].nubo", "style.css"} {
		file := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(file, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}

	router := New(root)
	if err := router.Reload(); err != nil {
		t.Fatal(err)
	}

	return router
}

func TestConcurrentReloadAndMatch(t *testing.T) {
	router := testRouter(t)
	start := make(chan struct{})
	done := make(chan error, 8)

	for worker := range 8 {
		go func() {
			<-start

			for range 30 {
				if worker < 4 {
					if err := router.Reload(); err != nil {
						done <- err
						return
					}

					continue
				}

				for _, path := range []string{"/", "/users/new", "/users/alice", "/style.css"} {
					route, ok := router.Match(path)
					if !ok {
						done <- fmt.Errorf("route %q disappeared during reload", path)
						return
					}

					if path == "/users/new" && route.Params["id"] != "" {
						done <- fmt.Errorf("exact route lost priority")
						return
					}

					if path == "/users/alice" && route.Params["id"] != "alice" {
						done <- fmt.Errorf("dynamic route lost its parameter")
						return
					}

					if route.IsExecutable != (path != "/style.css") {
						done <- fmt.Errorf("route %q changed its executable flag", path)
						return
					}
				}
			}

			done <- nil
		}()
	}

	close(start)
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	for range 8 {
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-timer.C:
			t.Fatal("concurrent router operations did not finish")
		}
	}
}

func TestFailedReloadPreservesRoutes(t *testing.T) {
	router := testRouter(t)
	if err := os.Rename(router.root, router.root+"-offline"); err != nil {
		t.Fatal(err)
	}

	if err := router.Reload(); err == nil {
		t.Fatal("reload should fail when its directory is missing")
	}

	if route, ok := router.Match("/users/alice"); !ok || route.Params["id"] != "alice" {
		t.Fatal("failed reload discarded the previous routes")
	}
}

func TestReloadReplacesRoutes(t *testing.T) {
	router := testRouter(t)
	if err := os.Remove(filepath.Join(router.root, "style.css")); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(router.root, "new.nubo"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	if err := router.Reload(); err != nil {
		t.Fatal(err)
	}

	if _, ok := router.Match("/style.css"); ok {
		t.Fatal("successful reload kept a removed route")
	}

	if _, ok := router.Match("/new"); !ok {
		t.Fatal("successful reload did not publish the new route")
	}
}
