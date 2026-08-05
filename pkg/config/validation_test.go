package config

import "testing"

func TestValidateSuccess(t *testing.T) {

	cfg := Default()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got: %v", err)
	}
}

func TestValidateEmptyProjectName(t *testing.T) {

	cfg := Default()
	cfg.Project.Name = ""

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateInvalidPort(t *testing.T) {

	cfg := Default()
	cfg.Server.Port = 70000

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateInvalidEnvironment(t *testing.T) {

	cfg := Default()
	cfg.Runtime.Environment = "local"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateInvalidLogLevel(t *testing.T) {

	cfg := Default()
	cfg.Runtime.LogLevel = "trace"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
