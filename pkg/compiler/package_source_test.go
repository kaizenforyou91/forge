package compiler

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewPackageSourceRegistry(t *testing.T) {
	registry := NewPackageSourceRegistry()

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	if registry.Count() != 0 {
		t.Fatalf("expected empty registry, got %d", registry.Count())
	}
}

func TestPackageSourceRegistryRegister(t *testing.T) {
	registry := NewPackageSourceRegistry()

	source := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	if err := registry.Register(source); err != nil {
		t.Fatal(err)
	}

	if registry.Count() != 1 {
		t.Fatalf("expected 1 source, got %d", registry.Count())
	}

	got, err := registry.Resolve("compiler", "v1")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, source) {
		t.Fatalf(
			"expected %#v, got %#v",
			source,
			got,
		)
	}
}

func TestPackageSourceRegistryRegisterTrimsValues(t *testing.T) {
	registry := NewPackageSourceRegistry()

	source := PackageSource{
		Name:       " compiler ",
		Version:    " v1 ",
		ImportPath: " github.com/kaizenforyou91/forge/pkg/compiler ",
	}

	if err := registry.Register(source); err != nil {
		t.Fatal(err)
	}

	got, err := registry.Resolve("compiler", "v1")
	if err != nil {
		t.Fatal(err)
	}

	want := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"expected %#v, got %#v",
			want,
			got,
		)
	}
}

func TestPackageSourceRegistryRejectsInvalidSource(t *testing.T) {
	tests := []struct {
		name   string
		source PackageSource
	}{
		{
			name: "empty name",
			source: PackageSource{
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
			},
		},
		{
			name: "empty version",
			source: PackageSource{
				Name:       "compiler",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
			},
		},
		{
			name: "empty import path",
			source: PackageSource{
				Name:    "compiler",
				Version: "v1",
			},
		},
		{
			name: "whitespace import path",
			source: PackageSource{
				Name:       "compiler",
				Version:    "v1",
				ImportPath: "   ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewPackageSourceRegistry()

			err := registry.Register(tt.source)

			if !errors.Is(err, ErrInvalidPackageSource) {
				t.Fatalf(
					"expected ErrInvalidPackageSource, got %v",
					err,
				)
			}
		})
	}
}

func TestPackageSourceRegistryRejectsDuplicate(t *testing.T) {
	registry := NewPackageSourceRegistry()

	source := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	if err := registry.Register(source); err != nil {
		t.Fatal(err)
	}

	err := registry.Register(source)

	if !errors.Is(err, ErrDuplicatePackageSource) {
		t.Fatalf(
			"expected ErrDuplicatePackageSource, got %v",
			err,
		)
	}
}

func TestPackageSourceRegistryResolveMissing(t *testing.T) {
	registry := NewPackageSourceRegistry()

	_, err := registry.Resolve("missing", "v1")

	if !errors.Is(err, ErrPackageSourceNotFound) {
		t.Fatalf(
			"expected ErrPackageSourceNotFound, got %v",
			err,
		)
	}
}

func TestPackageSourceRegistryListPreservesRegistrationOrder(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	sources := []PackageSource{
		{
			Name:       "app",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
		},
		{
			Name:       "compiler",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
		},
		{
			Name:       "registry",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/registry",
		},
	}

	for _, source := range sources {
		if err := registry.Register(source); err != nil {
			t.Fatal(err)
		}
	}

	got := registry.List()

	if !reflect.DeepEqual(got, sources) {
		t.Fatalf(
			"expected %#v, got %#v",
			sources,
			got,
		)
	}
}

func TestPackageSourceRegistryListReturnsIndependentSnapshot(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	source := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	if err := registry.Register(source); err != nil {
		t.Fatal(err)
	}

	got := registry.List()
	got[0].Name = "mutated"

	resolved, err := registry.Resolve("compiler", "v1")
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Name != "compiler" {
		t.Fatalf(
			"registry was mutated through snapshot: %#v",
			resolved,
		)
	}
}

func TestPackageSourceRegistryNilReceiver(t *testing.T) {
	var registry *PackageSourceRegistry

	if err := registry.Register(PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}); !errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf(
			"expected ErrInvalidPackageSource, got %v",
			err,
		)
	}

	if _, err := registry.Resolve("compiler", "v1"); !errors.Is(
		err,
		ErrPackageSourceNotFound,
	) {
		t.Fatalf(
			"expected ErrPackageSourceNotFound, got %v",
			err,
		)
	}

	if got := registry.List(); got != nil {
		t.Fatalf("expected nil list, got %#v", got)
	}

	if got := registry.Count(); got != 0 {
		t.Fatalf("expected 0 count, got %d", got)
	}
}
