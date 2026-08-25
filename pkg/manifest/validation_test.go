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

func TestManifestValidateAllowsAbsentEntrypoint(t *testing.T) {
	m := validTestManifest()
	m.Entrypoint = nil

	if err := m.Validate(); err != nil {
		t.Fatalf("expected absent entrypoint to be valid, got %v", err)
	}
}

func TestManifestValidateAcceptsExactApplicationEntrypoint(t *testing.T) {
	m := validTestManifest()
	m.Entrypoint = &ApplicationEntrypoint{
		Module:  "http",
		Version: "v1",
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("expected exact entrypoint to be valid, got %v", err)
	}
}

func TestManifestValidateEntrypointFailures(t *testing.T) {
	tests := []struct {
		name       string
		entrypoint ApplicationEntrypoint
		empty      bool
		want       string
	}{
		{
			name: "empty object",
			want: "manifest.entrypoint.module is required",
		},
		{
			name:       "module only",
			entrypoint: ApplicationEntrypoint{Module: "http"},
			want:       "manifest.entrypoint.version is required",
		},
		{
			name:       "version only",
			entrypoint: ApplicationEntrypoint{Version: "v1"},
			want:       "manifest.entrypoint.module is required",
		},
		{
			name:       "whitespace module",
			entrypoint: ApplicationEntrypoint{Module: "   ", Version: "v1"},
			want:       "manifest.entrypoint.module is required",
		},
		{
			name:       "whitespace version",
			entrypoint: ApplicationEntrypoint{Module: "http", Version: "   "},
			want:       "manifest.entrypoint.version is required",
		},
		{
			name:       "leading module whitespace",
			entrypoint: ApplicationEntrypoint{Module: " http", Version: "v1"},
			want:       "manifest.entrypoint.module must not contain surrounding whitespace",
		},
		{
			name:       "trailing module whitespace",
			entrypoint: ApplicationEntrypoint{Module: "http ", Version: "v1"},
			want:       "manifest.entrypoint.module must not contain surrounding whitespace",
		},
		{
			name:       "leading version whitespace",
			entrypoint: ApplicationEntrypoint{Module: "http", Version: " v1"},
			want:       "manifest.entrypoint.version must not contain surrounding whitespace",
		},
		{
			name:       "trailing version whitespace",
			entrypoint: ApplicationEntrypoint{Module: "http", Version: "v1 "},
			want:       "manifest.entrypoint.version must not contain surrounding whitespace",
		},
		{
			name:       "unknown module",
			entrypoint: ApplicationEntrypoint{Module: "missing", Version: "v1"},
			want:       `manifest.entrypoint references unknown module "missing"@"v1"`,
		},
		{
			name:       "wrong module version",
			entrypoint: ApplicationEntrypoint{Module: "http", Version: "v2"},
			want:       `manifest.entrypoint references unknown module "http"@"v2"`,
		},
		{
			name:       "entrypoint with empty modules",
			entrypoint: ApplicationEntrypoint{Module: "http", Version: "v1"},
			empty:      true,
			want:       `manifest.entrypoint references unknown module "http"@"v1"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := validTestManifest()
			m.Entrypoint = &test.entrypoint
			if test.empty {
				m.Modules = nil
			}

			err := m.Validate()
			if err == nil {
				t.Fatal("expected entrypoint validation error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
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
		})
	}
}

func TestManifestValidateEntrypointDoesNotRequireImportPath(t *testing.T) {
	m := validTestManifest()
	m.Modules[0].ImportPath = ""
	m.Entrypoint = &ApplicationEntrypoint{
		Module:  "http",
		Version: "v1",
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("expected pure manifest validation to allow empty import path, got %v", err)
	}
}

func TestManifestValidateExistingModuleErrorPrecedesEntrypointError(t *testing.T) {
	m := validTestManifest()
	m.Modules[0].Name = ""
	m.Entrypoint = &ApplicationEntrypoint{}

	err := m.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "manifest.modules[0].name is required") {
		t.Fatalf("expected existing module error to win, got %v", err)
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
