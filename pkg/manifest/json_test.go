package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSON(t *testing.T) {
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
      "version": "v1"
    }
  ]
}`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
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

func TestLoadJSONApplicationEntrypoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.json")

	data := []byte(`{
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

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
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

func TestLoadJSONNullApplicationEntrypointIsAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.json")

	data := []byte(`{
  "version": "v1",
  "name": "demo",
  "entrypoint": null,
  "modules": []
}`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
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

func TestLoadJSONPreservesModuleOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "forge.json")

	data := []byte(`{
  "version": "v1",
  "name": "ordered",
  "modules": [
    {
      "name": "first",
      "version": "v1"
    },
    {
      "name": "second",
      "version": "v1"
    },
    {
      "name": "third",
      "version": "v1"
    }
  ]
}`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"first", "second", "third"}

	if len(got.Modules) != len(want) {
		t.Fatalf(
			"expected %d modules, got %d",
			len(want),
			len(got.Modules),
		)
	}

	for i, name := range want {
		if got.Modules[i].Name != name {
			t.Fatalf(
				"module %d: expected %q, got %q",
				i,
				name,
				got.Modules[i].Name,
			)
		}
	}
}

func TestLoadJSONMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	if _, err := LoadJSON(path); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestLoadJSONInvalidSyntax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.json")

	data := []byte(`{
  "version": "v1",
  "name": "broken",
  "modules": [
`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadJSON(path); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestLoadJSONEmptyDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)

	if err == nil {
		t.Fatal("expected empty JSON document to return an error")
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
