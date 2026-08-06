package config

import "testing"

func TestOverrideHost(t *testing.T) {
	t.Setenv("FORGE_SERVER_HOST", "0.0.0.0")

	cfg := Default()

	if err := cfg.OverrideFromEnv(); err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected host 0.0.0.0, got %s", cfg.Server.Host)
	}
}

func TestOverridePort(t *testing.T) {
	t.Setenv("FORGE_SERVER_PORT", "9090")

	cfg := Default()

	if err := cfg.OverrideFromEnv(); err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
}

func TestOverrideLoggerLevel(t *testing.T) {
	t.Setenv("FORGE_LOGGER_LEVEL", "debug")

	cfg := Default()

	if err := cfg.OverrideFromEnv(); err != nil {
		t.Fatal(err)
	}

	if cfg.Runtime.LogLevel != "debug" {
		t.Fatalf("expected runtime loglevel debug, got %s", cfg.Runtime.LogLevel)
	}
}

func TestOverrideInvalidPort(t *testing.T) {
	t.Setenv("FORGE_SERVER_PORT", "abc")

	cfg := Default()

	if err := cfg.OverrideFromEnv(); err == nil {
		t.Fatal("expected error, got nil")
	}
}
