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

func TestManifestValidateAcceptsExactUnicodeIdentities(t *testing.T) {
	m := Manifest{
		Version: "versi-é",
		Name:    "aplikasi-東京",
		Entrypoint: &ApplicationEntrypoint{
			Module:  "aplikasi",
			Version: "vé",
		},
		Modules: []Module{
			{
				Name:    "aplikasi",
				Version: "vé",
				Dependencies: []Dependency{
					{Name: "pustaka", Version: "v一"},
				},
			},
			{Name: "pustaka", Version: "v一"},
		},
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("expected Unicode identities to remain valid, got %v", err)
	}
}

func TestManifestValidateRejectsAmbiguousIdentityComponents(t *testing.T) {
	invalidValues := []struct {
		name  string
		value string
	}{
		{name: "leading ASCII space", value: " identity"},
		{name: "trailing ASCII space", value: "identity "},
		{name: "leading Unicode whitespace", value: "\u2003identity"},
		{name: "trailing Unicode whitespace", value: "identity\u3000"},
		{name: "newline", value: "iden\ntity"},
		{name: "tab", value: "iden\ttity"},
		{name: "NUL", value: "iden\x00tity"},
		{name: "DEL", value: "iden\x7ftity"},
		{name: "at delimiter", value: "iden@tity"},
		{name: "invalid UTF-8", value: string([]byte{'i', 0xff})},
	}
	fields := []struct {
		name   string
		mutate func(*Manifest, string)
	}{
		{name: "manifest name", mutate: func(m *Manifest, value string) { m.Name = value }},
		{name: "manifest version", mutate: func(m *Manifest, value string) { m.Version = value }},
		{name: "module name", mutate: func(m *Manifest, value string) { m.Modules[0].Name = value }},
		{name: "module version", mutate: func(m *Manifest, value string) { m.Modules[0].Version = value }},
		{name: "dependency name", mutate: func(m *Manifest, value string) { m.Modules[0].Dependencies[0].Name = value }},
		{name: "dependency version", mutate: func(m *Manifest, value string) { m.Modules[0].Dependencies[0].Version = value }},
		{name: "entrypoint module", mutate: func(m *Manifest, value string) { m.Entrypoint.Module = value }},
		{name: "entrypoint version", mutate: func(m *Manifest, value string) { m.Entrypoint.Version = value }},
	}

	for _, field := range fields {
		for _, invalid := range invalidValues {
			t.Run(field.name+"/"+invalid.name, func(t *testing.T) {
				m := manifestIdentityTestManifest()
				field.mutate(&m, invalid.value)
				err := m.Validate()
				if err == nil {
					t.Fatal("expected identity validation error")
				}
				forgeErr, ok := err.(*forgeerrors.Error)
				if !ok || forgeErr.Code != forgeerrors.CodeInvalidManifest {
					t.Fatalf("expected invalid manifest error, got %T: %v", err, err)
				}
			})
		}
	}
}

func TestManifestValidateKeepsCanonicallyDifferentUnicodeExact(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "é", Version: "v1"},
			{Name: "e\u0301", Version: "v1"},
		},
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("expected canonically different exact identities, got %v", err)
	}
	if m.Modules[0].Name == m.Modules[1].Name {
		t.Fatal("test fixture identities unexpectedly compare equal")
	}
}

func TestManifestValidateDoesNotApplyIdentityRulesToImportPath(t *testing.T) {
	m := validTestManifest()
	m.Modules[0].ImportPath = " \t@import\x00path "

	if err := m.Validate(); err != nil {
		t.Fatalf("manifest validation unexpectedly changed ImportPath semantics: %v", err)
	}
}

func manifestIdentityTestManifest() Manifest {
	return Manifest{
		Version: "v1",
		Name:    "demo",
		Entrypoint: &ApplicationEntrypoint{
			Module:  "app",
			Version: "v1",
		},
		Modules: []Module{
			{
				Name:    "app",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "dep", Version: "v1"},
				},
			},
			{Name: "dep", Version: "v1"},
		},
	}
}
