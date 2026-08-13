package plugin

import (
	"testing"

	"github.com/kaizenforyou91/forge/pkg/app"
)

type testPlugin struct{}

var _ Plugin = (*testPlugin)(nil)

func (p *testPlugin) Name() string {
	return "test"
}

func (p *testPlugin) Version() string {
	return "1.0.0"
}

func (p *testPlugin) Register(a *app.App) error {
	return nil
}

func (p *testPlugin) Start(a *app.App) error {
	return nil
}

func (p *testPlugin) Stop(a *app.App) error {
	return nil
}

func TestPluginContract(t *testing.T) {
	var p Plugin = &testPlugin{}

	if p.Name() != "test" {
		t.Fatalf("expected plugin name %q, got %q", "test", p.Name())
	}

	if p.Version() != "1.0.0" {
		t.Fatalf("expected plugin version %q, got %q", "1.0.0", p.Version())
	}
}

func TestPluginImplementsModule(t *testing.T) {
	var p Plugin = &testPlugin{}
	var module app.Module = p

	if module.Name() != "test" {
		t.Fatalf("expected module name %q, got %q", "test", module.Name())
	}
}
