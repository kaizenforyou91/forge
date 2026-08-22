package registry

import (
	"errors"
	"reflect"
	"sync"
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	r := New()

	pkg := Package{
		Name:    "http",
		Version: "v1",
	}

	if err := r.Register(pkg); err != nil {
		t.Fatal(err)
	}

	got, err := r.Get("http", "v1")
	if err != nil {
		t.Fatal(err)
	}

	if got != pkg {
		t.Fatalf("expected %#v, got %#v", pkg, got)
	}

	if r.Count() != 1 {
		t.Fatalf("expected count 1, got %d", r.Count())
	}
}

func TestRegisterRejectsEmptyName(t *testing.T) {
	r := New()

	err := r.Register(Package{
		Version: "v1",
	})

	if !errors.Is(err, ErrInvalidPackage) {
		t.Fatalf("expected ErrInvalidPackage, got %v", err)
	}
}

func TestRegisterRejectsEmptyVersion(t *testing.T) {
	r := New()

	err := r.Register(Package{
		Name: "http",
	})

	if !errors.Is(err, ErrInvalidPackage) {
		t.Fatalf("expected ErrInvalidPackage, got %v", err)
	}
}

func TestRegisterRejectsDuplicatePackage(t *testing.T) {
	r := New()

	pkg := Package{
		Name:    "http",
		Version: "v1",
	}

	if err := r.Register(pkg); err != nil {
		t.Fatal(err)
	}

	if err := r.Register(pkg); !errors.Is(err, ErrDuplicatePackage) {
		t.Fatalf("expected ErrDuplicatePackage, got %v", err)
	}
}

func TestRegistryEnsureAllRegistersBatch(t *testing.T) {
	r := New()

	packages := []Package{
		{Name: "app", Version: "v1"},
		{Name: "compiler", Version: "v1"},
		{Name: "registry", Version: "v1"},
	}

	if err := r.EnsureAll(packages); err != nil {
		t.Fatal(err)
	}

	if r.Count() != len(packages) {
		t.Fatalf("expected count %d, got %d", len(packages), r.Count())
	}

	if got := r.List(); !reflect.DeepEqual(got, packages) {
		t.Fatalf("expected %#v, got %#v", packages, got)
	}
}

func TestRegistryEnsureAllRejectsInvalidPackageWithoutPartialRegistration(
	t *testing.T,
) {
	r := New()

	err := r.EnsureAll([]Package{
		{Name: "app", Version: "v1"},
		{Name: "compiler"},
		{Name: "registry", Version: "v1"},
	})

	if !errors.Is(err, ErrInvalidPackage) {
		t.Fatalf("expected ErrInvalidPackage, got %v", err)
	}

	if r.Count() != 0 {
		t.Fatalf("expected count 0, got %d", r.Count())
	}

	if got := r.List(); len(got) != 0 {
		t.Fatalf("expected empty list, got %#v", got)
	}
}

func TestRegistryEnsureAllPreservesExistingPackages(t *testing.T) {
	r := New()

	a := Package{Name: "app", Version: "v1"}
	b := Package{Name: "compiler", Version: "v1"}

	if err := r.Register(a); err != nil {
		t.Fatal(err)
	}

	if err := r.EnsureAll([]Package{a, b}); err != nil {
		t.Fatal(err)
	}

	want := []Package{a, b}
	if got := r.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestRegistryEnsureAllDeduplicatesInputIdentity(t *testing.T) {
	r := New()

	a := Package{Name: "app", Version: "v1"}
	b := Package{Name: "compiler", Version: "v1"}
	c := Package{Name: "registry", Version: "v1"}

	if err := r.EnsureAll([]Package{a, b, a, c, b}); err != nil {
		t.Fatal(err)
	}

	want := []Package{a, b, c}
	if got := r.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}

	if r.Count() != len(want) {
		t.Fatalf("expected count %d, got %d", len(want), r.Count())
	}
}

func TestRegistryEnsureAllAppendsNewPackagesInInputOrder(t *testing.T) {
	r := New()

	a := Package{Name: "app", Version: "v1"}
	b := Package{Name: "compiler", Version: "v1"}
	c := Package{Name: "registry", Version: "v1"}
	d := Package{Name: "worker", Version: "v1"}

	if err := r.Register(a); err != nil {
		t.Fatal(err)
	}

	if err := r.EnsureAll([]Package{c, b, a, d}); err != nil {
		t.Fatal(err)
	}

	want := []Package{a, c, b, d}
	if got := r.List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestRegistryEnsureAllIsIdempotent(t *testing.T) {
	r := New()

	packages := []Package{
		{Name: "app", Version: "v1"},
		{Name: "compiler", Version: "v1"},
		{Name: "registry", Version: "v1"},
	}

	for i := 0; i < 3; i++ {
		if err := r.EnsureAll(packages); err != nil {
			t.Fatal(err)
		}
	}

	if r.Count() != len(packages) {
		t.Fatalf("expected count %d, got %d", len(packages), r.Count())
	}

	if got := r.List(); !reflect.DeepEqual(got, packages) {
		t.Fatalf("expected %#v, got %#v", packages, got)
	}
}

func TestRegistryEnsureAllNilReceiver(t *testing.T) {
	var r *Registry

	err := r.EnsureAll([]Package{{Name: "app", Version: "v1"}})

	if !errors.Is(err, ErrInvalidPackage) {
		t.Fatalf("expected ErrInvalidPackage, got %v", err)
	}
}

func TestRegistryConcurrentEnsureAllIsAtomic(t *testing.T) {
	r := New()

	a := Package{Name: "app", Version: "v1"}
	b := Package{Name: "compiler", Version: "v1"}
	c := Package{Name: "registry", Version: "v1"}
	d := Package{Name: "worker", Version: "v1"}
	e := Package{Name: "logger", Version: "v1"}

	batches := [][]Package{
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

		go func(batch []Package) {
			defer wg.Done()
			<-start

			errorsCh <- r.EnsureAll(batch)
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

	expected := map[string]Package{
		packageKey(a.Name, a.Version): a,
		packageKey(b.Name, b.Version): b,
		packageKey(c.Name, c.Version): c,
		packageKey(d.Name, d.Version): d,
		packageKey(e.Name, e.Version): e,
	}

	if r.Count() != len(expected) {
		t.Fatalf("expected count %d, got %d", len(expected), r.Count())
	}

	seen := make(map[string]int, len(expected))
	for _, pkg := range r.List() {
		key := packageKey(pkg.Name, pkg.Version)
		want, ok := expected[key]
		if !ok {
			t.Fatalf("unexpected package %#v", pkg)
		}

		if pkg != want {
			t.Fatalf("expected %#v, got %#v", want, pkg)
		}

		seen[key]++
	}

	for key := range expected {
		if seen[key] != 1 {
			t.Fatalf("expected %s exactly once, got %d", key, seen[key])
		}
	}
}

func TestSameNameDifferentVersionIsAllowed(t *testing.T) {
	r := New()

	if err := r.Register(Package{
		Name:    "http",
		Version: "v1",
	}); err != nil {
		t.Fatal(err)
	}

	if err := r.Register(Package{
		Name:    "http",
		Version: "v2",
	}); err != nil {
		t.Fatal(err)
	}

	if r.Count() != 2 {
		t.Fatalf("expected count 2, got %d", r.Count())
	}
}

func TestGetMissingPackage(t *testing.T) {
	r := New()

	_, err := r.Get("missing", "v1")

	if !errors.Is(err, ErrPackageNotFound) {
		t.Fatalf("expected ErrPackageNotFound, got %v", err)
	}
}

func TestListPreservesRegistrationOrder(t *testing.T) {
	r := New()

	packages := []Package{
		{Name: "http", Version: "v1"},
		{Name: "worker", Version: "v1"},
		{Name: "logger", Version: "v2"},
	}

	for _, pkg := range packages {
		if err := r.Register(pkg); err != nil {
			t.Fatal(err)
		}
	}

	got := r.List()

	if len(got) != len(packages) {
		t.Fatalf("expected %d packages, got %d", len(packages), len(got))
	}

	for i := range packages {
		if got[i] != packages[i] {
			t.Fatalf("index %d: expected %#v, got %#v", i, packages[i], got[i])
		}
	}
}

func TestListReturnsSnapshot(t *testing.T) {
	r := New()

	if err := r.Register(Package{
		Name:    "http",
		Version: "v1",
	}); err != nil {
		t.Fatal(err)
	}

	snapshot := r.List()

	snapshot[0] = Package{
		Name:    "mutated",
		Version: "v9",
	}

	got, err := r.Get("http", "v1")
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "http" || got.Version != "v1" {
		t.Fatalf("registry was mutated through snapshot: %#v", got)
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := New()

	const writers = 20
	const readers = 20

	var wg sync.WaitGroup

	for i := 0; i < writers; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			_ = r.Register(Package{
				Name:    "pkg",
				Version: "v" + string(rune('a'+i)),
			})
		}(i)
	}

	for i := 0; i < readers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			_ = r.List()
			_, _ = r.Get("pkg", "unknown")
			_ = r.Count()
		}()
	}

	wg.Wait()

	if r.Count() != writers {
		t.Fatalf("expected %d packages, got %d", writers, r.Count())
	}
}
