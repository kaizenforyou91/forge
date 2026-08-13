package manifest

import (
	"strings"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

func validTestManifest() Manifest {
	return Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}
}

func TestManifestValidateSuccess(t *testing.T) {
	m := validTestManifest()

	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid manifest, got %v", err)
	}
}

func TestManifestValidateRequiresVersion(t *testing.T) {
	m := validTestManifest()
	m.Version = ""

	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "manifest.version is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateRequiresName(t *testing.T) {
	m := validTestManifest()
	m.Name = ""

	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "manifest.name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateRequiresModuleName(t *testing.T) {
	m := validTestManifest()
	m.Modules[0].Name = ""

	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "manifest.modules[0].name is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateRequiresModuleVersion(t *testing.T) {
	m := validTestManifest()
	m.Modules[0].Version = ""

	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "manifest.modules[0].version is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateRejectsDuplicateModules(t *testing.T) {
	m := validTestManifest()
	m.Modules = append(m.Modules, Module{
		Name:    "http",
		Version: "v2",
	})

	err := m.Validate()
	if err == nil {
		t.Fatal("expected duplicate module error")
	}

	if !strings.Contains(err.Error(), "duplicate module") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestManifestValidateAllowsEmptyModules(t *testing.T) {
	m := validTestManifest()
	m.Modules = nil

	if err := m.Validate(); err != nil {
		t.Fatalf("expected empty modules to be valid, got %v", err)
	}
}

func TestManifestValidateUsesInvalidManifestCode(t *testing.T) {
	m := validTestManifest()
	m.Name = ""

	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}

	forgeErr, ok := err.(*forgeerrors.Error)
	if !ok {
		t.Fatalf("expected *errors.Error, got %T", err)
	}

	if forgeErr.Code != forgeerrors.CodeInvalidManifest {
		t.Fatalf(
			"expected code %s, got %s",
			forgeerrors.CodeInvalidManifest,
			forgeErr.Code,
		)
	}
}
