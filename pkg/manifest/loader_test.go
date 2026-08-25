package manifest

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.yaml")

	data := []byte(`version: v1
name: demo
modules:
  - name: http
    version: v1
  - name: worker
    version: v1
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.Version != "v1" {
		t.Fatalf("expected version v1, got %q", got.Version)
	}

	if got.Name != "demo" {
		t.Fatalf("expected name demo, got %q", got.Name)
	}

	if got.Entrypoint != nil {
		t.Fatalf("expected absent entrypoint, got %#v", got.Entrypoint)
	}

	if len(got.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(got.Modules))
	}

	if got.Modules[0].Name != "http" {
		t.Fatalf("expected first module http, got %q", got.Modules[0].Name)
	}

	if got.Modules[1].Name != "worker" {
		t.Fatalf("expected second module worker, got %q", got.Modules[1].Name)
	}
}

func TestLoadYAMLApplicationEntrypoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.yaml")

	data := []byte(`version: v1
name: demo
entrypoint:
  module: app
  version: v1
modules:
  - name: app
    version: v1
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entrypoint == nil {
		t.Fatal("expected decoded application entrypoint")
	}
	if *got.Entrypoint != (ApplicationEntrypoint{Module: "app", Version: "v1"}) {
		t.Fatalf("unexpected entrypoint %#v", got.Entrypoint)
	}
}

func TestLoadYAMLNullApplicationEntrypointIsAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.yaml")

	data := []byte(`version: v1
name: demo
entrypoint: null
modules: []
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Entrypoint != nil {
		t.Fatalf("expected null entrypoint to decode as absent, got %#v", got.Entrypoint)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("expected null entrypoint to behave as absent, got %v", err)
	}
}

func TestLoadYAMLAndJSONApplicationEntrypointParity(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "forge.yaml")
	jsonPath := filepath.Join(dir, "forge.json")

	yamlData := []byte(`version: v1
name: demo
entrypoint:
  module: app
  version: v1
modules:
  - name: app
    version: v1
`)
	jsonData := []byte(`{
  "version": "v1",
  "name": "demo",
  "entrypoint": {
    "module": "app",
    "version": "v1"
  },
  "modules": [
    {
      "name": "app",
      "version": "v1"
    }
  ]
}`)

	if err := os.WriteFile(yamlPath, yamlData, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		t.Fatal(err)
	}

	yamlManifest, err := LoadYAML(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	jsonManifest, err := LoadJSON(jsonPath)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(yamlManifest, jsonManifest) {
		t.Fatalf(
			"expected equivalent YAML and JSON manifests, got\nYAML: %#v\nJSON: %#v",
			yamlManifest,
			jsonManifest,
		)
	}
}

func TestLoadYAMLPreservesModuleOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.yaml")

	data := []byte(`version: v1
name: ordered
modules:
  - name: first
    version: v1
  - name: second
    version: v1
  - name: third
    version: v1
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"first", "second", "third"}

	if len(got.Modules) != len(want) {
		t.Fatalf("expected %d modules, got %d", len(want), len(got.Modules))
	}

	for i, name := range want {
		if got.Modules[i].Name != name {
			t.Fatalf("module %d: expected %q, got %q", i, name, got.Modules[i].Name)
		}
	}
}

func TestLoadYAMLMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.yaml")

	if _, err := LoadYAML(path); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestLoadYAMLInvalidSyntax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")

	if err := os.WriteFile(path, []byte("version: [unclosed"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadYAML(path); err == nil {
		t.Fatal("expected invalid YAML error")
	}
}

func TestLoadYAMLEmptyDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")

	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.Version != "" {
		t.Fatalf("expected empty version, got %q", got.Version)
	}

	if got.Name != "" {
		t.Fatalf("expected empty name, got %q", got.Name)
	}

	if len(got.Modules) != 0 {
		t.Fatalf("expected no modules, got %d", len(got.Modules))
	}
}
