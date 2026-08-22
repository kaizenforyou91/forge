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

func TestPackageSourceRegistryEnsureAllRegistersBatch(t *testing.T) {
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

	if err := registry.EnsureAll(sources); err != nil {
		t.Fatal(err)
	}

	if registry.Count() != len(sources) {
		t.Fatalf(
			"expected %d sources, got %d",
			len(sources),
			registry.Count(),
		)
	}

	if got := registry.List(); !reflect.DeepEqual(got, sources) {
		t.Fatalf("expected %#v, got %#v", sources, got)
	}
}

func TestPackageSourceRegistryEnsureAllRejectsInvalidSourceWithoutPartialRegistration(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	err := registry.EnsureAll([]PackageSource{
		{
			Name:       "app",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
		},
		{
			Name:       "compiler",
			Version:    "v1",
			ImportPath: "   ",
		},
		{
			Name:       "registry",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/registry",
		},
	})

	if !errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
	}

	if registry.Count() != 0 {
		t.Fatalf("expected 0 sources, got %d", registry.Count())
	}

	if got := registry.List(); len(got) != 0 {
		t.Fatalf("expected empty list, got %#v", got)
	}
}

func TestPackageSourceRegistryEnsureAllRejectsLaterConflictWithoutPartialRegistration(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	canonical := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}
	newA := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	}
	newC := PackageSource{
		Name:       "registry",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/registry",
	}

	if err := registry.Register(canonical); err != nil {
		t.Fatal(err)
	}

	before := registry.List()
	err := registry.EnsureAll([]PackageSource{
		newA,
		{
			Name:       canonical.Name,
			Version:    canonical.Version,
			ImportPath: "github.com/kaizenforyou91/forge/pkg/conflict",
		},
		newC,
	})

	if !errors.Is(err, ErrPackageSourceConflict) {
		t.Fatalf("expected ErrPackageSourceConflict, got %v", err)
	}

	for _, source := range []PackageSource{newA, newC} {
		_, resolveErr := registry.Resolve(source.Name, source.Version)
		if !errors.Is(resolveErr, ErrPackageSourceNotFound) {
			t.Fatalf(
				"expected %s@%s to remain absent, got %v",
				source.Name,
				source.Version,
				resolveErr,
			)
		}
	}

	got, resolveErr := registry.Resolve(canonical.Name, canonical.Version)
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}

	if !reflect.DeepEqual(got, canonical) {
		t.Fatalf("expected %#v, got %#v", canonical, got)
	}

	if registry.Count() != 1 {
		t.Fatalf("expected 1 source, got %d", registry.Count())
	}

	if got := registry.List(); !reflect.DeepEqual(got, before) {
		t.Fatalf("expected unchanged list %#v, got %#v", before, got)
	}
}

func TestPackageSourceRegistryEnsureAllRejectsIntraBatchConflict(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	err := registry.EnsureAll([]PackageSource{
		{
			Name:       "app",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/path-one",
		},
		{
			Name:       "app",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/path-two",
		},
	})

	if !errors.Is(err, ErrPackageSourceConflict) {
		t.Fatalf("expected ErrPackageSourceConflict, got %v", err)
	}

	for _, fragment := range []string{
		"app@v1",
		"github.com/kaizenforyou91/forge/path-one",
		"github.com/kaizenforyou91/forge/path-two",
	} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("expected error to contain %q, got %v", fragment, err)
		}
	}

	if registry.Count() != 0 {
		t.Fatalf("expected 0 sources, got %d", registry.Count())
	}

	if got := registry.List(); len(got) != 0 {
		t.Fatalf("expected empty list, got %#v", got)
	}
}

func TestPackageSourceRegistryEnsureAllAcceptsIntraBatchIdenticalDuplicate(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	source := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	}

	if err := registry.EnsureAll([]PackageSource{
		source,
		source,
		source,
	}); err != nil {
		t.Fatal(err)
	}

	want := []PackageSource{source}
	if got := registry.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}

	if registry.Count() != 1 {
		t.Fatalf("expected 1 source, got %d", registry.Count())
	}
}

func TestPackageSourceRegistryEnsureAllNormalizesBeforeComparison(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	want := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	}

	if err := registry.EnsureAll([]PackageSource{
		{
			Name:       " app ",
			Version:    " v1 ",
			ImportPath: " github.com/kaizenforyou91/forge/pkg/app ",
		},
		want,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := registry.Resolve(want.Name, want.Version)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}

	if registry.Count() != 1 {
		t.Fatalf("expected 1 source, got %d", registry.Count())
	}
}

func TestPackageSourceRegistryEnsureAllPreservesCanonicalSources(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	canonical := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	}
	newSource := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}

	if err := registry.Register(canonical); err != nil {
		t.Fatal(err)
	}

	if err := registry.EnsureAll([]PackageSource{
		canonical,
		newSource,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := registry.Resolve(canonical.Name, canonical.Version)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, canonical) {
		t.Fatalf("expected %#v, got %#v", canonical, got)
	}

	want := []PackageSource{canonical, newSource}
	if got := registry.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestPackageSourceRegistryEnsureAllAppendsNewSourcesInInputOrder(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	a := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	}
	b := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}
	c := PackageSource{
		Name:       "registry",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/registry",
	}
	d := PackageSource{
		Name:       "worker",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/worker",
	}

	if err := registry.Register(a); err != nil {
		t.Fatal(err)
	}

	if err := registry.EnsureAll([]PackageSource{c, b, a, d}); err != nil {
		t.Fatal(err)
	}

	want := []PackageSource{a, c, b, d}
	if got := registry.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestPackageSourceRegistryEnsureAllIsIdempotent(t *testing.T) {
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

	for i := 0; i < 3; i++ {
		if err := registry.EnsureAll(sources); err != nil {
			t.Fatal(err)
		}
	}

	if registry.Count() != len(sources) {
		t.Fatalf(
			"expected %d sources, got %d",
			len(sources),
			registry.Count(),
		)
	}

	if got := registry.List(); !reflect.DeepEqual(got, sources) {
		t.Fatalf("expected %#v, got %#v", sources, got)
	}
}

func TestPackageSourceRegistryEnsureAllNilReceiver(t *testing.T) {
	var registry *PackageSourceRegistry

	err := registry.EnsureAll([]PackageSource{
		{
			Name:       "app",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
		},
	})

	if !errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
	}
}

func TestPackageSourceRegistryConcurrentEnsureAllIsAtomic(t *testing.T) {
	registry := NewPackageSourceRegistry()

	a := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	}
	b := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}
	c := PackageSource{
		Name:       "registry",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/registry",
	}
	d := PackageSource{
		Name:       "worker",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/worker",
	}
	e := PackageSource{
		Name:       "logger",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/logger",
	}

	batches := [][]PackageSource{
		{a, b, c},
		{b, c, d},
		{d, e, a},
	}

	const workers = 32

	errorsCh := make(chan error, workers)
	start := make(chan struct{})

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		batch := batches[i%len(batches)]
		wg.Add(1)

		go func(batch []PackageSource) {
			defer wg.Done()
			<-start

			errorsCh <- registry.EnsureAll(batch)
		}(batch)
	}

	close(start)
	wg.Wait()
	close(errorsCh)

	for err := range errorsCh {
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	}

	expected := map[string]PackageSource{
		packageSourceKey(a.Name, a.Version): a,
		packageSourceKey(b.Name, b.Version): b,
		packageSourceKey(c.Name, c.Version): c,
		packageSourceKey(d.Name, d.Version): d,
		packageSourceKey(e.Name, e.Version): e,
	}

	if registry.Count() != len(expected) {
		t.Fatalf(
			"expected %d sources, got %d",
			len(expected),
			registry.Count(),
		)
	}

	seen := make(map[string]int, len(expected))
	for _, source := range registry.List() {
		key := packageSourceKey(source.Name, source.Version)
		want, ok := expected[key]
		if !ok {
			t.Fatalf("unexpected source %#v", source)
		}

		if !reflect.DeepEqual(source, want) {
			t.Fatalf("expected %#v, got %#v", want, source)
		}

		seen[key]++
	}

	for key := range expected {
		if seen[key] != 1 {
			t.Fatalf("expected %s exactly once, got %d", key, seen[key])
		}
	}
}

func TestPackageSourceRegistryConcurrentEnsureAllPreservesCanonicalConflict(
	t *testing.T,
) {
	registry := NewPackageSourceRegistry()

	canonical := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
	}
	b := PackageSource{
		Name:       "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}
	c := PackageSource{
		Name:       "registry",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/registry",
	}
	rejectedD := PackageSource{
		Name:       "worker",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/worker",
	}
	rejectedE := PackageSource{
		Name:       "logger",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/logger",
	}
	conflicting := PackageSource{
		Name:       canonical.Name,
		Version:    canonical.Version,
		ImportPath: "github.com/kaizenforyou91/forge/pkg/conflict",
	}

	if err := registry.Ensure(canonical); err != nil {
		t.Fatal(err)
	}

	successBatch := []PackageSource{canonical, b, c}
	conflictBatch := []PackageSource{rejectedD, conflicting, rejectedE}

	type ensureAllResult struct {
		wantConflict bool
		err          error
	}

	const workers = 32

	results := make(chan ensureAllResult, workers)
	start := make(chan struct{})

	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		batch := successBatch
		wantConflict := i%2 != 0
		if wantConflict {
			batch = conflictBatch
		}

		wg.Add(1)

		go func(batch []PackageSource, wantConflict bool) {
			defer wg.Done()
			<-start

			results <- ensureAllResult{
				wantConflict: wantConflict,
				err:          registry.EnsureAll(batch),
			}
		}(batch, wantConflict)
	}

	close(start)
	wg.Wait()
	close(results)

	for result := range results {
		if result.wantConflict {
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

	got, err := registry.Resolve(canonical.Name, canonical.Version)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, canonical) {
		t.Fatalf("expected %#v, got %#v", canonical, got)
	}

	for _, source := range []PackageSource{b, c} {
		got, resolveErr := registry.Resolve(source.Name, source.Version)
		if resolveErr != nil {
			t.Fatal(resolveErr)
		}

		if !reflect.DeepEqual(got, source) {
			t.Fatalf("expected %#v, got %#v", source, got)
		}
	}

	for _, source := range []PackageSource{rejectedD, rejectedE} {
		_, resolveErr := registry.Resolve(source.Name, source.Version)
		if !errors.Is(resolveErr, ErrPackageSourceNotFound) {
			t.Fatalf(
				"expected %s@%s to remain absent, got %v",
				source.Name,
				source.Version,
				resolveErr,
			)
		}
	}

	if registry.Count() != 3 {
		t.Fatalf("expected 3 sources, got %d", registry.Count())
	}

	seen := make(map[string]int, registry.Count())
	for _, source := range registry.List() {
		seen[packageSourceKey(source.Name, source.Version)]++
	}

	for _, source := range []PackageSource{canonical, b, c} {
		key := packageSourceKey(source.Name, source.Version)
		if seen[key] != 1 {
			t.Fatalf("expected %s exactly once, got %d", key, seen[key])
		}
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
