package manifest

import "testing"

func TestManifestContract(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}

	if m.Version != "v1" {
		t.Fatalf("unexpected manifest version: %q", m.Version)
	}

	if m.Name != "demo" {
		t.Fatalf("unexpected manifest name: %q", m.Name)
	}

	if len(m.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(m.Modules))
	}

	if m.Modules[0].Name != "http" {
		t.Fatalf("unexpected module name: %q", m.Modules[0].Name)
	}
}
