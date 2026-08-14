package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestYAMLManifestPipeline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.yaml")

	data := []byte(`version: v1
name: demo
modules:
  - name: http
    version: v1
  - name: worker
    version: v2
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("load YAML: %v", err)
	}

	if err := manifest.Validate(); err != nil {
		t.Fatalf("validate YAML manifest: %v", err)
	}

	resolved, err := Resolve(manifest, []Module{
		{Name: "worker", Version: "v2"},
		{Name: "http", Version: "v1"},
	})
	if err != nil {
		t.Fatalf("resolve YAML manifest: %v", err)
	}

	if resolved.Name != "demo" {
		t.Fatalf("expected manifest name %q, got %q", "demo", resolved.Name)
	}

	if len(resolved.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(resolved.Modules))
	}

	if resolved.Modules[0].Name != "http" ||
		resolved.Modules[0].Version != "v1" {
		t.Fatalf("unexpected first resolved module: %#v", resolved.Modules[0])
	}

	if resolved.Modules[1].Name != "worker" ||
		resolved.Modules[1].Version != "v2" {
		t.Fatalf("unexpected second resolved module: %#v", resolved.Modules[1])
	}
}

func TestJSONManifestPipeline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.json")

	data := []byte(`{
  "version": "v1",
  "name": "demo",
  "modules": [
    {
      "name": "http",
      "version": "v1"
    },
    {
      "name": "worker",
      "version": "v2"
    }
  ]
}`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadJSON(path)
	if err != nil {
		t.Fatalf("load JSON: %v", err)
	}

	if err := manifest.Validate(); err != nil {
		t.Fatalf("validate JSON manifest: %v", err)
	}

	resolved, err := Resolve(manifest, []Module{
		{Name: "http", Version: "v1"},
		{Name: "worker", Version: "v2"},
	})
	if err != nil {
		t.Fatalf("resolve JSON manifest: %v", err)
	}

	if resolved.Name != "demo" {
		t.Fatalf("expected manifest name %q, got %q", "demo", resolved.Name)
	}

	if len(resolved.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(resolved.Modules))
	}
}

func TestManifestPipelineRejectsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")

	data := []byte(`version: v1
name:
modules:
  - name: http
    version: v1
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadYAML(path)
	if err != nil {
		t.Fatalf("unexpected YAML load error: %v", err)
	}

	if err := manifest.Validate(); err == nil {
		t.Fatal("expected manifest validation error")
	}
}

func TestManifestPipelineRejectsMissingModule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.json")

	data := []byte(`{
  "version": "v1",
  "name": "demo",
  "modules": [
    {
      "name": "http",
      "version": "v2"
    }
  ]
}`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadJSON(path)
	if err != nil {
		t.Fatalf("load JSON: %v", err)
	}

	if err := manifest.Validate(); err != nil {
		t.Fatalf("validate manifest: %v", err)
	}

	if _, err := Resolve(manifest, []Module{
		{Name: "http", Version: "v1"},
	}); err == nil {
		t.Fatal("expected module resolution error")
	}
}

func TestResolveDoesNotMutateInput(t *testing.T) {
	manifest := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v1"},
		},
	}

	original := manifest.Modules[0]

	_, err := Resolve(manifest, []Module{
		{Name: "http", Version: "v1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if manifest.Modules[0] != original {
		t.Fatal("Resolve mutated the input manifest")
	}
}
