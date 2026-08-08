package http

import (
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/middleware"
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

func TestRouterMiddlewareIntegration(t *testing.T) {
	r := router.New()

	var order []string

	logging := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "middleware-before")

			next.ServeHTTP(w, req)

			order = append(order, "middleware-after")
		})
	}

	r.GET("/hello/:name", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		order = append(order, "handler")

		name := router.Param(req, "name")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello " + name))
	}))

	handler := middleware.Chain(logging)(r)

	module := NewModule("127.0.0.1:0", handler)

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

	resp, err := http.Get(
		"http://" + module.Host().ListenerAddr().String() + "/hello/forge",
	)
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

	expectedOrder := []string{
		"middleware-before",
		"handler",
		"middleware-after",
	}

	if !reflect.DeepEqual(order, expectedOrder) {
		t.Fatalf("unexpected execution order: %#v", order)
	}
}

func TestModuleMiddlewarePipeline(t *testing.T) {
	var order []string

	first := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "first-before")

			next.ServeHTTP(w, req)

			order = append(order, "first-after")
		})
	}

	second := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "second-before")

			next.ServeHTTP(w, req)

			order = append(order, "second-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		order = append(order, "handler")

		w.WriteHeader(http.StatusOK)
	})

	module := NewModule(
		"127.0.0.1:0",
		handler,
		first,
		second,
	)

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

	resp, err := http.Get(
		"http://" + module.Host().ListenerAddr().String() + "/",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	expected := []string{
		"first-before",
		"second-before",
		"handler",
		"second-after",
		"first-after",
	}

	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("unexpected middleware order: %#v", order)
	}
}

func TestModuleMiddlewareShortCircuit(t *testing.T) {
	handlerCalled := false

	blocking := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Error(w, "blocked", http.StatusForbidden)
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	module := NewModule(
		"127.0.0.1:0",
		handler,
		blocking,
	)

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

	resp, err := http.Get(
		"http://" + module.Host().ListenerAddr().String() + "/",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf(
			"expected status 403, got %d",
			resp.StatusCode,
		)
	}

	if handlerCalled {
		t.Fatal("handler should not be called")
	}
}
