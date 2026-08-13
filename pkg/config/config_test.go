package config

import "testing"

func TestDefaultConfig(t *testing.T) {

	cfg := Default()

	if cfg.Project.Name == "" {
		t.Fatal("project name should not be empty")
	}

	if cfg.Server.Port != 8080 {
		t.Fatal("unexpected default port")
	}

	if !cfg.Plugins.Logger.Enabled {
		t.Fatal("logger should be enabled")
	}
}

func TestPluginsIsEnabled(t *testing.T) {
	cfg := Default()

	if !cfg.Plugins.IsEnabled("logger") {
		t.Fatal("expected logger plugin to be enabled by default")
	}

	cfg.Plugins.Logger.Enabled = false

	if cfg.Plugins.IsEnabled("logger") {
		t.Fatal("expected logger plugin to be disabled")
	}

	if cfg.Plugins.IsEnabled("unknown") {
		t.Fatal("expected unknown plugin to be disabled")
	}
}
