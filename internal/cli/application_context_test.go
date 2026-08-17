package cli

import (
	"testing"

	"github.com/kaizenforyou91/forge/internal/bootstrap"
	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestNewRootCommandCreatesApplicationContext(t *testing.T) {
	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)

	if cmd == nil {
		t.Fatal("expected root command")
	}

	got, err := ApplicationFromContext(cmd.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got != application {
		t.Fatal("expected root command to retain injected application")
	}

	if !got.HasModule("compiler") {
		t.Fatal("expected compiler module")
	}
}

func TestApplicationFromContextProvidesCompilerModule(t *testing.T) {
	application := bootstrap.NewApplication()
	cmd := NewRootCommandWithApplication(application)

	got, err := ApplicationFromContext(cmd.Context())
	if err != nil {
		t.Fatal(err)
	}

	var engine *compiler.Engine

	if err := got.Container().Resolve(&engine); err != nil {
		t.Fatal(err)
	}

	if engine == nil {
		t.Fatal("expected compiler engine")
	}
}
