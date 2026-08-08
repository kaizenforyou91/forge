package http

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/router"
)

func TestModule(t *testing.T) {
	module := NewModule("127.0.0.1:0", http.NewServeMux())

	if module == nil {
		t.Fatal("expected module")
	}

	if module.Name() != "http" {
		t.Fatalf("unexpected module name: %s", module.Name())
	}

	if module.Host() == nil {
		t.Fatal("expected host")
	}
}

func TestModuleLifecycle(t *testing.T) {
	a := app.New()

	module := NewModule("127.0.0.1:0", http.NewServeMux())

	a.Use(module)

	if err := a.Start(); err != nil {
		t.Fatal(err)
	}

	if !a.Started() {
		t.Fatal("application should be started")
	}

	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}

	if a.Started() {
		t.Fatal("application should be stopped")
	}
}

func TestModuleStartDoesNotBlock(t *testing.T) {
	module := NewModule("127.0.0.1:0", http.NewServeMux())

	a := app.New()

	if err := module.Register(a); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)

	go func() {
		done <- module.Start(a)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}

	case <-time.After(time.Second):
		t.Fatal("module start blocked")
	}

	ctxDone := make(chan struct{})

	go func() {
		_ = module.Stop(a)
		close(ctxDone)
	}()

	select {
	case <-ctxDone:
	case <-time.After(time.Second):
		t.Fatal("module stop blocked")
	}
}

func TestRouterIntegration(t *testing.T) {
	r := router.New()

	r.GET("/hello", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello forge"))
	}))

	module := NewRouterModule("127.0.0.1:0", r)

	if module == nil {
		t.Fatal("expected module")
	}

	if module.Host() == nil {
		t.Fatal("expected host")
	}

	if module.Host().Handler() != r {
		t.Fatal("expected router to be HTTP handler")
	}
}

func TestRouterIntegrationRequest(t *testing.T) {
	r := router.New()

	r.GET("/hello/:name", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		name := router.Param(req, "name")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello " + name))
	}))

	module := NewRouterModule("127.0.0.1:0", r)

	a := app.New()

	a.Use(module)

	if err := a.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := a.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	resp, err := http.Get("http://" + module.Host().ListenerAddr().String() + "/hello/forge")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "hello forge" {
		t.Fatalf("unexpected response: %q", string(body))
	}
}
