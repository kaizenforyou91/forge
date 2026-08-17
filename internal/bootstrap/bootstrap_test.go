package bootstrap

import (
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestNewApplicationRegistersCompilerModule(t *testing.T) {
	application := NewApplication()

	if application == nil {
		t.Fatal("expected application")
	}

	if !application.HasModule("compiler") {
		t.Fatal("expected compiler module")
	}

	if len(application.Modules()) != 1 {
		t.Fatalf(
			"expected 1 module, got %d",
			len(application.Modules()),
		)
	}

	module := application.Modules()[0]

	compilerModule, ok := module.(*compiler.Module)
	if !ok {
		t.Fatalf(
			"expected *compiler.Module, got %T",
			module,
		)
	}

	if compilerModule.Engine() == nil {
		t.Fatal("expected compiler engine")
	}

	if compilerModule.Registry() == nil {
		t.Fatal("expected compiler registry")
	}
}

func TestNewApplicationCanRun(t *testing.T) {
	application := NewApplication()

	if err := application.Run(); err != nil {
		t.Fatal(err)
	}

	if !application.Started() {
		t.Fatal("expected application to be started")
	}

	if err := application.Stop(); err != nil {
		t.Fatal(err)
	}
}
