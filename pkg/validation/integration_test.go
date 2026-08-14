package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

func TestYAMLManifestValidationIntegration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.yaml")

	data := []byte(`version: v1
name: demo
modules:
  - name: http
    version: v1
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	m, err := manifest.LoadYAML(path)
	if err != nil {
		t.Fatalf("load YAML: %v", err)
	}

	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Validate(m); err != nil {
		t.Fatalf("expected valid manifest, got %v", err)
	}
}

func TestJSONManifestValidationIntegration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.json")

	data := []byte(`{
  "version": "v1",
  "name": "demo",
  "modules": [
    {
      "name": "http",
      "version": "v1"
    }
  ]
}`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	m, err := manifest.LoadJSON(path)
	if err != nil {
		t.Fatalf("load JSON: %v", err)
	}

	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Validate(m); err != nil {
		t.Fatalf("expected valid manifest, got %v", err)
	}
}

func TestValidationEngineRejectsInvalidLoadedManifest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.yaml")

	data := []byte(`version: v1
name:
modules:
  - name: http
    version: v1
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	m, err := manifest.LoadYAML(path)
	if err != nil {
		t.Fatalf("load YAML: %v", err)
	}

	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Validate(m); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidationEngineRunsSemanticValidatorAfterStructuralValidation(t *testing.T) {
	engine, err := NewEngine(
		ValidatorFunc(func(m manifest.Manifest) error {
			if m.Name != "demo" {
				t.Fatalf("unexpected manifest name %q", m.Name)
			}
			return nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
	}

	if err := engine.Validate(m); err != nil {
		t.Fatalf("expected validation success, got %v", err)
	}
}
