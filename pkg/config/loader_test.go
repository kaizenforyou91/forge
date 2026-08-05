package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {

	content := `
project:
  name: test-app
`

	file := "test-forge.yaml"

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	defer os.Remove(file)

	cfg, err := Load(file)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Project.Name != "test-app" {
		t.Fatal("config not loaded")
	}
}
