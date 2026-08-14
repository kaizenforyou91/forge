package registry

import (
	"errors"
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
