package config

import (
	"os"
	"testing"
)

func TestManager(t *testing.T) {

	tmp, err := os.CreateTemp("", "*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	tmp.WriteString(`
project:
  name: Forge
  version: dev

runtime:
  environment: development
  log_level: info

server:
  host: localhost
  port: 8080

plugins:
  logger:
    enabled: true
`)

	tmp.Close()

	manager, err := NewManager(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}

	cfg := manager.Config()

	if cfg.Server.Port != 8080 {
		t.Fatal("manager failed")
	}
}
func TestManagerSave(t *testing.T) {

	cfg := Default()

	file := "manager.yaml"
	defer os.Remove(file)

	if err := Save(file, cfg); err != nil {
		t.Fatal(err)
	}

	mgr, err := NewManager(file)
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.Save(); err != nil {
		t.Fatal(err)
	}
}
