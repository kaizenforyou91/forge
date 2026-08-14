package manifest

import (
	"errors"
	"reflect"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

func TestResolveDependencyOrder(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
					{Name: "logger", Version: "v1"},
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

	got, err := ResolveDependencyOrder(m)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"logger@v1",
		"http@v1",
		"web@v1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %#v, got %#v", want, got)
	}
}

func TestResolveDependencyOrderPreservesDeterministicTieBreak(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
					{Name: "http", Version: "v1"},
				},
			},
			{
				Name:    "logger",
				Version: "v1",
			},
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}

	got, err := ResolveDependencyOrder(m)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"logger@v1",
		"http@v1",
		"web@v1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected deterministic order %#v, got %#v", want, got)
	}
}

func TestResolveDependencyOrderSharedDependencyOnlyOnce(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "worker",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	got, err := ResolveDependencyOrder(m)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"http@v1",
		"logger@v1",
		"web@v1",
		"worker@v1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %#v, got %#v", want, got)
	}
}

func TestResolveDependencyOrderRejectsCycle(t *testing.T) {
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

	_, err := ResolveDependencyOrder(m)
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
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

func TestResolveDependencyOrderRejectsMissingDependency(t *testing.T) {
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
		},
	}

	_, err := ResolveDependencyOrder(m)
	if err == nil {
		t.Fatal("expected missing dependency error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T", err)
	}

	if forgeErr.Code != forgeerrors.CodeNotFound {
		t.Fatalf(
			"expected code %s, got %s",
			forgeerrors.CodeNotFound,
			forgeErr.Code,
		)
	}
}

func TestResolveDependencyOrderEmptyManifest(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "empty",
	}

	got, err := ResolveDependencyOrder(m)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 0 {
		t.Fatalf("expected empty order, got %#v", got)
	}
}

func TestResolveDependencyOrderDoesNotMutateInput(t *testing.T) {
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

	_, err := ResolveDependencyOrder(m)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m, original) {
		t.Fatal("ResolveDependencyOrder mutated input manifest")
	}
}
