package manifest

import (
	"strings"
	"testing"
)

func TestManifestDependencyContract(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{
						Name:    "http",
						Version: "v1",
					},
				},
			},
		},
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid dependency contract, got %v", err)
	}

	if len(m.Modules[0].Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(m.Modules[0].Dependencies))
	}

	dependency := m.Modules[0].Dependencies[0]

	if dependency.Name != "http" {
		t.Fatalf("unexpected dependency name: %q", dependency.Name)
	}

	if dependency.Version != "v1" {
		t.Fatalf("unexpected dependency version: %q", dependency.Version)
	}
}

func TestManifestValidateRequiresDependencyName(t *testing.T) {
	m := validTestManifest()
	m.Modules[0].Dependencies = []Dependency{
		{
			Version: "v1",
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(
		err.Error(),
		"manifest.modules[0].dependencies[0].name is required",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateRequiresDependencyVersion(t *testing.T) {
	m := validTestManifest()
	m.Modules[0].Dependencies = []Dependency{
		{
			Name: "http",
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(
		err.Error(),
		"manifest.modules[0].dependencies[0].version is required",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateRejectsDuplicateDependencies(t *testing.T) {
	m := validTestManifest()

	m.Modules[0] = Module{
		Name:    "web",
		Version: "v1",
		Dependencies: []Dependency{
			{Name: "http", Version: "v1"},
			{Name: "http", Version: "v1"},
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected duplicate dependency error")
	}

	if !strings.Contains(err.Error(), "duplicate dependency") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateRejectsSelfDependency(t *testing.T) {
	m := validTestManifest()

	m.Modules[0].Dependencies = []Dependency{
		{
			Name:    m.Modules[0].Name,
			Version: m.Modules[0].Version,
		},
	}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected self-dependency error")
	}

	if !strings.Contains(err.Error(), "cannot depend on itself") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateAllowsEmptyDependencies(t *testing.T) {
	m := validTestManifest()
	m.Modules[0].Dependencies = nil

	if err := m.Validate(); err != nil {
		t.Fatalf("expected empty dependencies to be valid, got %v", err)
	}
}

func TestManifestDependencyYAMLAndJSONTags(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
				},
			},
		},
	}

	if m.Modules[0].Dependencies[0].Name != "http" {
		t.Fatal("dependency name was not stored correctly")
	}

	if m.Modules[0].Dependencies[0].Version != "v1" {
		t.Fatal("dependency version was not stored correctly")
	}
}
