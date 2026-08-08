package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/app"
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
