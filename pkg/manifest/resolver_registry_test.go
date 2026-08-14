package manifest

import (
	"errors"
	"reflect"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

func testRegistry(t *testing.T) *registry.Registry {
	t.Helper()

	r := registry.New()

	packages := []registry.Package{
		{Name: "http", Version: "v1"},
		{Name: "worker", Version: "v2"},
	}

	for _, pkg := range packages {
		if err := r.Register(pkg); err != nil {
			t.Fatal(err)
		}
	}

	return r
}

func TestResolveFromRegistry(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v1"},
			{Name: "worker", Version: "v2"},
		},
	}

	got, err := ResolveFromRegistry(m, testRegistry(t))
	if err != nil {
		t.Fatal(err)
	}

	if got.Version != m.Version {
		t.Fatalf("expected manifest version %q, got %q", m.Version, got.Version)
	}

	if got.Name != m.Name {
		t.Fatalf("expected manifest name %q, got %q", m.Name, got.Name)
	}

	if len(got.Modules) != 2 {
		t.Fatalf("expected 2 resolved modules, got %d", len(got.Modules))
	}

	if !reflect.DeepEqual(got.Modules[0], m.Modules[0]) {
		t.Fatalf("unexpected first module: %#v", got.Modules[0])
	}

	if !reflect.DeepEqual(got.Modules[1], m.Modules[1]) {
		t.Fatalf("unexpected second module: %#v", got.Modules[1])
	}
}

func TestResolveFromRegistryUsesExactVersion(t *testing.T) {
	r := registry.New()

	if err := r.Register(registry.Package{
		Name:    "http",
		Version: "v1",
	}); err != nil {
		t.Fatal(err)
	}

	if err := r.Register(registry.Package{
		Name:    "http",
		Version: "v2",
	}); err != nil {
		t.Fatal(err)
	}

	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v2"},
		},
	}

	got, err := ResolveFromRegistry(m, r)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(got.Modules))
	}

	if got.Modules[0].Version != "v2" {
		t.Fatalf(
			"expected exact version %q, got %q",
			"v2",
			got.Modules[0].Version,
		)
	}
}

func TestResolveFromRegistryPreservesManifestOrder(t *testing.T) {
	r := registry.New()

	for _, pkg := range []registry.Package{
		{Name: "second", Version: "v2"},
		{Name: "first", Version: "v1"},
		{Name: "third", Version: "v1"},
	} {
		if err := r.Register(pkg); err != nil {
			t.Fatal(err)
		}
	}

	m := Manifest{
		Version: "v1",
		Name:    "ordered",
		Modules: []Module{
			{Name: "third", Version: "v1"},
			{Name: "first", Version: "v1"},
			{Name: "second", Version: "v2"},
		},
	}

	got, err := ResolveFromRegistry(m, r)
	if err != nil {
		t.Fatal(err)
	}

	for i, want := range m.Modules {
		if !reflect.DeepEqual(got.Modules[i], want) {
			t.Fatalf(
				"module %d: expected %q@%q, got %q@%q",
				i,
				want.Name,
				want.Version,
				got.Modules[i].Name,
				got.Modules[i].Version,
			)
		}
	}
}

func TestResolveFromRegistryMissingPackage(t *testing.T) {
	r := registry.New()

	if err := r.Register(registry.Package{
		Name:    "http",
		Version: "v1",
	}); err != nil {
		t.Fatal(err)
	}

	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v2"},
		},
	}

	_, err := ResolveFromRegistry(m, r)
	if err == nil {
		t.Fatal("expected resolution error")
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

	if !errors.Is(err, registry.ErrPackageNotFound) {
		t.Fatalf("expected wrapped registry error, got %v", err)
	}
}

func TestResolveFromRegistryRejectsInvalidManifest(t *testing.T) {
	m := Manifest{
		Version: "",
		Name:    "demo",
	}

	_, err := ResolveFromRegistry(m, testRegistry(t))
	if err == nil {
		t.Fatal("expected validation error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T", err)
	}

	if forgeErr.Code != forgeerrors.CodeInvalidManifest {
		t.Fatalf(
			"expected error code %s, got %s",
			forgeerrors.CodeInvalidManifest,
			forgeErr.Code,
		)
	}
}

func TestResolveFromRegistryRejectsNilRegistry(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
	}

	_, err := ResolveFromRegistry(m, nil)
	if err == nil {
		t.Fatal("expected nil registry error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T", err)
	}

	if forgeErr.Code != forgeerrors.CodeInternal {
		t.Fatalf(
			"expected error code %s, got %s",
			forgeerrors.CodeInternal,
			forgeErr.Code,
		)
	}
}

func TestResolveFromRegistryDoesNotMutateInput(t *testing.T) {
	r := testRegistry(t)

	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v1"},
		},
	}

	original := append([]Module(nil), m.Modules...)

	_, err := ResolveFromRegistry(m, r)
	if err != nil {
		t.Fatal(err)
	}

	if len(m.Modules) != len(original) {
		t.Fatalf(
			"manifest module count changed from %d to %d",
			len(original),
			len(m.Modules),
		)
	}

	for i := range original {
		if !reflect.DeepEqual(m.Modules[i], original[i]) {
			t.Fatalf(
				"input manifest mutated at index %d",
				i,
			)
		}
	}
}

func TestResolveRemainsBackwardCompatible(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v1"},
		},
	}

	got, err := Resolve(m, []Module{
		{Name: "http", Version: "v1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(got.Modules))
	}

	if !reflect.DeepEqual(got.Modules[0], m.Modules[0]) {
		t.Fatalf(
			"unexpected resolved module: %#v",
			got.Modules[0],
		)
	}
}
