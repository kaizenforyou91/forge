package manifest

import (
	"errors"
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

func TestLoadYAMLAllowsOmittedOptionalCollections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.yaml")
	if err := os.WriteFile(path, []byte("version: v1\nname: demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Modules != nil || got.Entrypoint != nil {
		t.Fatalf("expected omitted optional values to remain nil, got %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("expected canonical manifest with omitted modules, got %v", err)
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

	if _, err := LoadYAML(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestLoadYAMLInvalidSyntax(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")

	if err := os.WriteFile(path, []byte("version: [unclosed"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadYAML(path)
	requireInvalidManifestError(t, err)
}

func TestLoadYAMLEmptyDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")

	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadYAML(path)
	requireInvalidManifestError(t, err)
}

func TestLoadYAMLStrictPreAlphaContract(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "invalid UTF-8", data: append([]byte("version: v1\nname: "), append([]byte{0xff}, []byte("\nmodules: []\n")...)...)},
		{name: "UTF-8 BOM", data: append([]byte{0xef, 0xbb, 0xbf}, []byte("version: v1\nname: demo\nmodules: []\n")...)},
		{name: "UTF-16 little endian BOM", data: []byte{0xff, 0xfe, 'v', 0}},
		{name: "UTF-16 big endian BOM", data: []byte{0xfe, 0xff, 0, 'v'}},
		{name: "whitespace only", data: []byte(" \r\n\t")},
		{name: "comments only", data: []byte("# no manifest\n")},
		{name: "null root", data: []byte("null\n")},
		{name: "sequence root", data: []byte("- version\n- v1\n")},
		{name: "scalar root", data: []byte("hello\n")},
		{name: "unknown root field", data: []byte("version: v1\nname: demo\nmodules: []\nextra: true\n")},
		{name: "case variant root field", data: []byte("Version: v1\nname: demo\nmodules: []\n")},
		{name: "case variant module field", data: []byte("version: v1\nname: demo\nmodules: [{Name: app, version: v1}]\n")},
		{name: "unknown entrypoint field", data: []byte("version: v1\nname: demo\nentrypoint: {module: app, version: v1, extra: value}\nmodules: []\n")},
		{name: "unknown module field", data: []byte("version: v1\nname: demo\nmodules: [{name: app, version: v1, extra: value}]\n")},
		{name: "unknown dependency field", data: []byte("version: v1\nname: demo\nmodules: [{name: app, version: v1, dependencies: [{name: dep, version: v1, extra: value}]}]\n")},
		{name: "duplicate root field", data: []byte("version: v1\nversion: v2\nname: demo\nmodules: []\n")},
		{name: "duplicate entrypoint field", data: []byte("version: v1\nname: demo\nentrypoint: {module: app, module: other, version: v1}\nmodules: []\n")},
		{name: "duplicate module field", data: []byte("version: v1\nname: demo\nmodules: [{name: app, name: other, version: v1}]\n")},
		{name: "duplicate dependency field", data: []byte("version: v1\nname: demo\nmodules: [{name: app, version: v1, dependencies: [{name: dep, name: other, version: v1}]}]\n")},
		{name: "anchor", data: []byte("version: &identity v1\nname: demo\nmodules: []\n")},
		{name: "alias", data: []byte("version: &identity v1\nname: demo\nmodules: []\nother: *identity\n")},
		{name: "merge key", data: []byte("version: v1\nname: demo\nmodules:\n  - <<: {name: app, version: v1}\n")},
		{name: "explicit core tag", data: []byte("version: !!str v1\nname: demo\nmodules: []\n")},
		{name: "explicit local tag", data: []byte("version: !identity v1\nname: demo\nmodules: []\n")},
		{name: "explicit nonspecific tag", data: []byte("version: ! v1\nname: demo\nmodules: []\n")},
		{name: "unused tag directive", data: []byte("%TAG !example! tag:example.com,2026:\n---\nversion: v1\nname: demo\nmodules: []\n")},
		{name: "implicit integer string field", data: []byte("version: 1\nname: demo\nmodules: []\n")},
		{name: "implicit boolean string field", data: []byte("version: v1\nname: true\nmodules: []\n")},
		{name: "implicit float string field", data: []byte("version: v1\nname: 1.25\nmodules: []\n")},
		{name: "implicit timestamp string field", data: []byte("version: v1\nname: 2026-09-03\nmodules: []\n")},
		{name: "implicit nested scalar", data: []byte("version: v1\nname: demo\nmodules: [{name: app, version: 1}]\n")},
		{name: "non-string mapping key", data: []byte("version: v1\nname: demo\nmodules: []\n1: value\n")},
		{name: "wrong entrypoint shape", data: []byte("version: v1\nname: demo\nentrypoint: []\nmodules: []\n")},
		{name: "wrong modules shape", data: []byte("version: v1\nname: demo\nmodules: {}\n")},
		{name: "wrong module item shape", data: []byte("version: v1\nname: demo\nmodules: [app]\n")},
		{name: "wrong dependencies shape", data: []byte("version: v1\nname: demo\nmodules: [{name: app, version: v1, dependencies: {}}]\n")},
		{name: "populated second document", data: []byte("version: v1\nname: demo\nmodules: []\n---\nversion: v2\nname: other\nmodules: []\n")},
		{name: "empty second document", data: []byte("version: v1\nname: demo\nmodules: []\n---\n")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "forge.yaml")
			if err := os.WriteFile(path, test.data, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadYAML(path)
			requireInvalidManifestError(t, err)
		})
	}
}

func TestLoadYAMLAllowsSafeBoringForms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.yaml")
	data := []byte(`--- # one explicit document
version: 'v1'
name: "démø"
entrypoint: null
modules: [
  {name: app, version: v1, import_path: example.com/app, dependencies: []}
]
... # explicit end
# trailing comment
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "démø" || got.Entrypoint != nil || len(got.Modules) != 1 {
		t.Fatalf("unexpected decoded manifest %#v", got)
	}
}

func TestLoadYAMLAllowsBlockStringSyntaxAtDecodeBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.yaml")
	data := []byte("version: v1\nname: |-\n  demo\nmodules: []\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "demo" {
		t.Fatalf("expected block string value, got %q", got.Name)
	}
}

func TestLoadYAMLDoesNotTreatDirectiveTextInScalarsOrCommentsAsSyntax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.yaml")
	data := []byte(`# %TAG !example! tag:example.com,2026:
version: v1
name: |-
  %TAG !example! tag:example.com,2026:
modules: []
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "%TAG !example! tag:example.com,2026:" {
		t.Fatalf("unexpected block scalar %q", got.Name)
	}
}

func TestLoadYAMLDoesNotRunDomainValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.yaml")
	if err := os.WriteFile(path, []byte("modules: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadYAML(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := got.Validate(); err == nil {
		t.Fatal("expected separate domain validation to reject missing identity")
	}
}
