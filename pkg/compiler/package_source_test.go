package compiler

import (
	"errors"
	"reflect"
	"strings"
	"sync"
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

func TestPackageSourceRegistryEnsureIdenticalSourceIsIdempotent(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	source := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	for i := 0; i < 3; i++ {
		if err := registry.Ensure(source); err != nil {
			t.Fatal(err)
		}
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

func TestPackageSourceRegistryEnsureTrimsValues(t *testing.T) {
	registry := NewPackageSourceRegistry()

	if err := registry.Ensure(PackageSource{
		Name:       " compiler ",
		Version:    " v1 ",
		ImportPath: " github.com/kaizenforyou91/forge/pkg/compiler ",
	}); err != nil {
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

func TestPackageSourceRegistryEnsureRejectsInvalidSource(t *testing.T) {
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

			err := registry.Ensure(tt.source)

			if !errors.Is(err, ErrInvalidPackageSource) {
				t.Fatalf(
					"expected ErrInvalidPackageSource, got %v",
					err,
				)
			}
		})
	}
}

func TestPackageSourceRegistryEnsureRejectsConflictingImportPath(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	original := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	if err := registry.Ensure(original); err != nil {
		t.Fatal(err)
	}

	err := registry.Ensure(PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	})

	if !errors.Is(err, ErrPackageSourceConflict) {
		t.Fatalf(
			"expected ErrPackageSourceConflict, got %v",
			err,
		)
	}

	if !strings.Contains(err.Error(), "compiler@v1") {
		t.Fatalf("expected source identity in error, got %v", err)
	}

	got, resolveErr := registry.Resolve("compiler", "v1")
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}

	if got.ImportPath != original.ImportPath {
		t.Fatalf(
			"expected import path %q, got %q",
			original.ImportPath,
			got.ImportPath,
		)
	}

	if registry.Count() != 1 {
		t.Fatalf("expected 1 source, got %d", registry.Count())
	}
}

func TestPackageSourceRegistryEnsureDoesNotDuplicateListEntry(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	source := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	for i := 0; i < 3; i++ {
		if err := registry.Ensure(source); err != nil {
			t.Fatal(err)
		}
	}

	want := []PackageSource{source}
	got := registry.List()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"expected %#v, got %#v",
			want,
			got,
		)
	}
}

func TestPackageSourceRegistryEnsurePreservesRegistrationOrder(
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

	for _, source := range sources[:2] {
		if err := registry.Ensure(source); err != nil {
			t.Fatal(err)
		}
	}

	if err := registry.Ensure(sources[0]); err != nil {
		t.Fatal(err)
	}

	if err := registry.Ensure(sources[2]); err != nil {
		t.Fatal(err)
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

func TestPackageSourceRegistryConcurrentIdenticalEnsure(t *testing.T) {
	registry := NewPackageSourceRegistry()

	source := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	const workers = 32

	errorsCh := make(chan error, workers)

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			errorsCh <- registry.Ensure(source)
		}()
	}

	wg.Wait()
	close(errorsCh)

	for err := range errorsCh {
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	}

	if registry.Count() != 1 {
		t.Fatalf("expected 1 source, got %d", registry.Count())
	}

	if got := len(registry.List()); got != 1 {
		t.Fatalf("expected 1 listed source, got %d", got)
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

func TestPackageSourceRegistryConcurrentEnsureRejectsConflict(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	canonical := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	conflicting := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	}

	if err := registry.Ensure(canonical); err != nil {
		t.Fatal(err)
	}

	type ensureResult struct {
		conflict bool
		err      error
	}

	const workers = 32

	results := make(chan ensureResult, workers)

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		source := canonical
		wantConflict := i%2 != 0

		if wantConflict {
			source = conflicting
		}

		wg.Add(1)

		go func() {
			defer wg.Done()

			results <- ensureResult{
				conflict: wantConflict,
				err:      registry.Ensure(source),
			}
		}()
	}

	wg.Wait()
	close(results)

	for result := range results {
		if result.conflict {
			if !errors.Is(result.err, ErrPackageSourceConflict) {
				t.Fatalf(
					"expected ErrPackageSourceConflict, got %v",
					result.err,
				)
			}

			continue
		}

		if result.err != nil {
			t.Fatalf("expected nil error, got %v", result.err)
		}
	}

	got, err := registry.Resolve("compiler", "v1")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, canonical) {
		t.Fatalf(
			"expected %#v, got %#v",
			canonical,
			got,
		)
	}

	if registry.Count() != 1 {
		t.Fatalf("expected 1 source, got %d", registry.Count())
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

	source := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	if err := registry.Register(source); !errors.Is(
		err,
		ErrInvalidPackageSource,
	) {
		t.Fatalf(
			"expected ErrInvalidPackageSource, got %v",
			err,
		)
	}

	if err := registry.Ensure(source); !errors.Is(
		err,
		ErrInvalidPackageSource,
	) {
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
