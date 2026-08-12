package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigShowWithDefault(t *testing.T) {
	dir := t.TempDir()
	cmd := newConfigCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"show", "--config", filepath.Join(dir, "forge.yaml")})

	err := cmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out.Bytes(), []byte("Forge Configuration")) {
		t.Fatalf("expected config show output, got %q", out.String())
	}
}

func TestConfigValidateSuccess(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "forge.yaml")

	yaml := `project:
  name: Forge
  version: dev

runtime:
  environment: development
  log_level: info

server:
  host: 127.0.0.1
  port: 8080

plugins:
  logger:
    enabled: true
`

	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newConfigCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"validate", "--config", cfgPath})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out.Bytes(), []byte("Configuration valid.")) {
		t.Fatalf("expected validate success output, got %q", out.String())
	}
}

func TestConfigValidateFailure(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "forge.yaml")

	if err := os.WriteFile(cfgPath, []byte("invalid: [unclosed"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newConfigCmd()
	cmd.SetArgs([]string{"validate", "--config", cfgPath})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected validate failure")
	}
}

func TestConfigInitCreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "forge.yaml")

	cmd := newConfigCmd()
	cmd.SetArgs([]string{"init", "--config", cfgPath})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(cfgPath); err != nil {
		t.Fatal(err)
	}
}

func TestConfigDoctor(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "forge.yaml")

	yaml := `project:
  name: Forge
  version: dev

runtime:
  environment: development
  log_level: info

server:
  host: 127.0.0.1
  port: 8080

plugins:
  logger:
    enabled: true
`

	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newConfigCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"doctor", "--config", cfgPath})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out.Bytes(), []byte("Configuration Doctor")) {
		t.Fatalf("expected doctor output, got %q", out.String())
	}
}

func TestConfigFlagPath(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "custom.yaml")

	yaml := `project:
  name: Forge
  version: dev

runtime:
  environment: development
  log_level: info

server:
  host: 127.0.0.1
  port: 8080

plugins:
  logger:
    enabled: true
`

	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newConfigCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"show", "--config", cfgPath})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !bytes.Contains(out.Bytes(), []byte("Configuration file: "+cfgPath)) {
		t.Fatalf("expected custom config path output, got %q", out.String())
	}
}
