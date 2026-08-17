package compiler

import (
	"testing"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

func TestNewModule(t *testing.T) {
	module := NewModule()

	if module == nil {
		t.Fatal("expected non-nil compiler module")
	}

	if module.Registry() == nil {
		t.Fatal("expected registry")
	}

	if module.Runner() == nil {
		t.Fatal("expected command runner")
	}

	if module.Executor() == nil {
		t.Fatal("expected executor")
	}

	if module.Engine() == nil {
		t.Fatal("expected engine")
	}
}

func TestModuleName(t *testing.T) {
	module := NewModule()

	if got := module.Name(); got != "compiler" {
		t.Fatalf("expected compiler module name, got %q", got)
	}
}

func TestModuleRegister(t *testing.T) {
	a := app.New()
	module := NewModule()

	if err := module.Register(a); err != nil {
		t.Fatal(err)
	}

	var gotRegistry *registry.Registry
	if err := a.Container().Resolve(&gotRegistry); err != nil {
		t.Fatal(err)
	}

	if gotRegistry != module.Registry() {
		t.Fatal("expected registered registry to match module registry")
	}

	var gotRunner *OSCommandRunner
	if err := a.Container().Resolve(&gotRunner); err != nil {
		t.Fatal(err)
	}

	if gotRunner != module.Runner() {
		t.Fatal("expected registered runner to match module runner")
	}

	var gotExecutor *ToolchainExecutor
	if err := a.Container().Resolve(&gotExecutor); err != nil {
		t.Fatal(err)
	}

	if gotExecutor != module.Executor() {
		t.Fatal("expected registered executor to match module executor")
	}

	var gotEngine *Engine
	if err := a.Container().Resolve(&gotEngine); err != nil {
		t.Fatal(err)
	}

	if gotEngine != module.Engine() {
		t.Fatal("expected registered engine to match module engine")
	}
}

func TestModuleStartStopAreNoop(t *testing.T) {
	a := app.New()
	module := NewModule()

	if err := module.Start(a); err != nil {
		t.Fatalf("unexpected Start error: %v", err)
	}

	if err := module.Stop(a); err != nil {
		t.Fatalf("unexpected Stop error: %v", err)
	}
}

func TestModuleCanBeAddedToApp(t *testing.T) {
	a := app.New()
	module := NewModule()

	if err := a.Add(module); err != nil {
		t.Fatal(err)
	}

	if !a.HasModule("compiler") {
		t.Fatal("expected compiler module to be registered")
	}

	if len(a.Modules()) != 1 {
		t.Fatalf("expected 1 module, got %d", len(a.Modules()))
	}
}

func TestModuleLifecycle(t *testing.T) {
	a := app.New()
	module := NewModule()

	if err := a.Add(module); err != nil {
		t.Fatal(err)
	}

	if err := a.Start(); err != nil {
		t.Fatal(err)
	}

	if !a.Started() {
		t.Fatal("expected application to be running")
	}

	if err := a.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestModuleSourceRegistry(t *testing.T) {
	module := NewModule()

	if module.SourceRegistry() == nil {
		t.Fatal("expected source registry")
	}

	source, err := module.SourceRegistry().Resolve(
		"compiler",
		"v1",
	)
	if err != nil {
		t.Fatal(err)
	}

	if source.ImportPath !=
		"github.com/kaizenforyou91/forge/pkg/compiler" {
		t.Fatalf(
			"unexpected import path: %q",
			source.ImportPath,
		)
	}
}
