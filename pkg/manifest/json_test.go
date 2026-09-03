package manifest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
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

func TestLoadJSONAllowsOmittedOptionalCollections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(`{"version":"v1","name":"demo"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
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

	if _, err := LoadJSON(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
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

	_, err := LoadJSON(path)
	requireInvalidManifestError(t, err)
}

func TestLoadJSONStrictPreAlphaContract(t *testing.T) {
	canonical := []byte(`{"version":"v1","name":"demo","entrypoint":{"module":"app","version":"v1"},"modules":[{"name":"app","version":"v1","import_path":"example.com/app","dependencies":[{"name":"dep","version":"v1"}]},{"name":"dep","version":"v1"}]}`)
	tests := []struct {
		name string
		data []byte
	}{
		{name: "invalid UTF-8", data: append([]byte(`{"version":"v1","name":"`), append([]byte{0xff}, []byte(`","modules":[]}`)...)...)},
		{name: "UTF-8 BOM", data: append([]byte{0xef, 0xbb, 0xbf}, canonical...)},
		{name: "null root", data: []byte(`null`)},
		{name: "array root", data: []byte(`[]`)},
		{name: "string root", data: []byte(`"hello"`)},
		{name: "number root", data: []byte(`123`)},
		{name: "boolean root", data: []byte(`true`)},
		{name: "unknown root field", data: []byte(`{"version":"v1","name":"demo","modules":[],"extra":true}`)},
		{name: "case variant root field", data: []byte(`{"Version":"v1","name":"demo","modules":[]}`)},
		{name: "case variant module field", data: []byte(`{"version":"v1","name":"demo","modules":[{"name":"app","version":"v1","ImportPath":"example.com/app"}]}`)},
		{name: "unknown entrypoint field", data: []byte(`{"version":"v1","name":"demo","entrypoint":{"module":"app","version":"v1","extra":true},"modules":[]}`)},
		{name: "unknown module field", data: []byte(`{"version":"v1","name":"demo","modules":[{"name":"app","version":"v1","extra":true}]}`)},
		{name: "unknown dependency field", data: []byte(`{"version":"v1","name":"demo","modules":[{"name":"app","version":"v1","dependencies":[{"name":"dep","version":"v1","extra":true}]}]}`)},
		{name: "duplicate root field", data: []byte(`{"version":"v1","version":"v2","name":"demo","modules":[]}`)},
		{name: "duplicate entrypoint field", data: []byte(`{"version":"v1","name":"demo","entrypoint":{"module":"app","module":"other","version":"v1"},"modules":[]}`)},
		{name: "duplicate module field", data: []byte(`{"version":"v1","name":"demo","modules":[{"name":"app","name":"other","version":"v1"}]}`)},
		{name: "duplicate dependency field", data: []byte(`{"version":"v1","name":"demo","modules":[{"name":"app","version":"v1","dependencies":[{"name":"dep","name":"other","version":"v1"}]}]}`)},
		{name: "wrong string shape", data: []byte(`{"version":1,"name":"demo","modules":[]}`)},
		{name: "wrong entrypoint shape", data: []byte(`{"version":"v1","name":"demo","entrypoint":[],"modules":[]}`)},
		{name: "wrong modules shape", data: []byte(`{"version":"v1","name":"demo","modules":null}`)},
		{name: "wrong module item shape", data: []byte(`{"version":"v1","name":"demo","modules":["app"]}`)},
		{name: "wrong dependencies shape", data: []byte(`{"version":"v1","name":"demo","modules":[{"name":"app","version":"v1","dependencies":{}}]}`)},
		{name: "wrong dependency item shape", data: []byte(`{"version":"v1","name":"demo","modules":[{"name":"app","version":"v1","dependencies":["dep"]}]}`)},
		{name: "trailing object", data: append(append([]byte(nil), canonical...), []byte(` {}`)...)},
		{name: "trailing primitive", data: append(append([]byte(nil), canonical...), []byte(` true`)...)},
		{name: "trailing garbage", data: append(append([]byte(nil), canonical...), []byte(` garbage`)...)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "forge.json")
			if err := os.WriteFile(path, test.data, 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := LoadJSON(path)
			requireInvalidManifestError(t, err)
		})
	}
}

func TestLoadJSONAllowsTrailingWhitespaceAndLiteralReplacementRune(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	data := []byte("{\"version\":\"v1\",\"name\":\"demo�\",\"modules\":[]}\r\n\t ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "demo�" {
		t.Fatalf("expected literal replacement rune to remain intact, got %q", got.Name)
	}
}

func TestLoadJSONDoesNotRunDomainValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "forge.json")
	if err := os.WriteFile(path, []byte(`{"modules":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != "" || got.Name != "" {
		t.Fatalf("unexpected decoded manifest %#v", got)
	}
	if err := got.Validate(); err == nil {
		t.Fatal("expected separate domain validation to reject missing identity")
	}
}

func TestLoadJSONEmptyDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")

	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadJSON(path)

	requireInvalidManifestError(t, err)

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

func requireInvalidManifestError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected invalid manifest error")
	}
	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if forgeErr.Code != forgeerrors.CodeInvalidManifest {
		t.Fatalf("expected code %s, got %s", forgeerrors.CodeInvalidManifest, forgeErr.Code)
	}
}
