package manifest

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

func dependencyTestRegistry(t *testing.T, packages ...registry.Package) *registry.Registry {
	t.Helper()

	r := registry.New()

	for _, pkg := range packages {
		if err := r.Register(pkg); err != nil {
			t.Fatal(err)
		}
	}

	return r
}

func TestResolveDependencies(t *testing.T) {
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
			{
				Name:    "http",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
		registry.Package{Name: "logger", Version: "v1"},
	)

	got, err := ResolveDependencies(m, r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Fatalf("resolved manifest differs from input: %#v", got)
	}
}

func TestResolveDependenciesExactVersion(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v2"},
				},
			},
			{
				Name:    "http",
				Version: "v2",
			},
		},
	}

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
		registry.Package{Name: "http", Version: "v2"},
	)

	if _, err := ResolveDependencies(m, r); err != nil {
		t.Fatalf("expected exact version resolution to succeed: %v", err)
	}
}

func TestResolveDependenciesTransitive(t *testing.T) {
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
			{
				Name:    "http",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
		registry.Package{Name: "logger", Version: "v1"},
	)

	if _, err := ResolveDependencies(m, r); err != nil {
		t.Fatalf("expected transitive dependency resolution to succeed: %v", err)
	}
}

func TestResolveDependenciesMissingRoot(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
			},
		},
	}

	r := dependencyTestRegistry(t)

	_, err := ResolveDependencies(m, r)
	if err == nil {
		t.Fatal("expected missing root module error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T", err)
	}

	if forgeErr.Code != forgeerrors.CodeNotFound {
		t.Fatalf(
			"expected error code %s, got %s",
			forgeerrors.CodeNotFound,
			forgeErr.Code,
		)
	}
}

func TestResolveDependenciesMissingDirectDependency(t *testing.T) {
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
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
	)

	_, err := ResolveDependencies(m, r)
	if err == nil {
		t.Fatal("expected missing direct dependency error")
	}

	if !strings.Contains(err.Error(), `http@v1`) {
		t.Fatalf("expected missing dependency identity in error, got %v", err)
	}
}

func TestResolveDependenciesMissingTransitiveDependency(t *testing.T) {
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
			{
				Name:    "http",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
	)

	_, err := ResolveDependencies(m, r)
	if err == nil {
		t.Fatal("expected missing transitive dependency error")
	}

	if !strings.Contains(err.Error(), `logger@v1`) {
		t.Fatalf("expected missing transitive dependency identity in error, got %v", err)
	}
}

func TestResolveDependenciesPreservesManifestOrder(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "ordered",
		Modules: []Module{
			{Name: "third", Version: "v1"},
			{Name: "first", Version: "v1"},
			{Name: "second", Version: "v1"},
		},
	}

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "second", Version: "v1"},
		registry.Package{Name: "first", Version: "v1"},
		registry.Package{Name: "third", Version: "v1"},
	)

	got, err := ResolveDependencies(m, r)
	if err != nil {
		t.Fatal(err)
	}

	for i := range m.Modules {
		if !reflect.DeepEqual(got.Modules[i], m.Modules[i]) {
			t.Fatalf(
				"module %d order changed: expected %#v, got %#v",
				i,
				m.Modules[i],
				got.Modules[i],
			)
		}
	}
}

func TestResolveDependenciesPreservesDependencyMetadata(t *testing.T) {
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
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
	)

	got, err := ResolveDependencies(m, r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(
		got.Modules[0].Dependencies,
		m.Modules[0].Dependencies,
	) {
		t.Fatalf(
			"dependency metadata was lost: expected %#v, got %#v",
			m.Modules[0].Dependencies,
			got.Modules[0].Dependencies,
		)
	}
}

func TestResolveDependenciesRejectsCycle(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "cycle",
		Modules: []Module{
			{
				Name:    "a",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "b", Version: "v1"},
				},
			},
			{
				Name:    "b",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "a", Version: "v1"},
				},
			},
		},
	}

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "a", Version: "v1"},
		registry.Package{Name: "b", Version: "v1"},
	)

	_, err := ResolveDependencies(m, r)
	if err == nil {
		t.Fatal("expected cycle error")
	}

	if !strings.Contains(err.Error(), "circular module dependency detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveDependenciesRejectsNilRegistry(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
	}

	_, err := ResolveDependencies(m, nil)
	if err == nil {
		t.Fatal("expected nil registry error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T", err)
	}

	if forgeErr.Code != forgeerrors.CodeInternal {
		t.Fatalf(
			"expected code %s, got %s",
			forgeerrors.CodeInternal,
			forgeErr.Code,
		)
	}
}

func TestResolveDependenciesDoesNotMutateInput(t *testing.T) {
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
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}

	original := m

	r := dependencyTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
	)

	_, err := ResolveDependencies(m, r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m, original) {
		t.Fatal("ResolveDependencies mutated input manifest")
	}
}

func TestResolveDependenciesAllowsEmptyModules(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "empty",
	}

	r := dependencyTestRegistry(t)

	got, err := ResolveDependencies(m, r)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Modules) != 0 {
		t.Fatalf("expected 0 modules, got %d", len(got.Modules))
	}
}
