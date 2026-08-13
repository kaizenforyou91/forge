package manifest

import (
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

func TestResolveManifest(t *testing.T) {
	manifest := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v1"},
			{Name: "worker", Version: "v2"},
		},
	}

	available := []Module{
		{Name: "worker", Version: "v1"},
		{Name: "http", Version: "v1"},
		{Name: "worker", Version: "v2"},
	}

	got, err := Resolve(manifest, available)
	if err != nil {
		t.Fatal(err)
	}

	if got.Version != manifest.Version {
		t.Fatalf(
			"expected manifest version %q, got %q",
			manifest.Version,
			got.Version,
		)
	}

	if got.Name != manifest.Name {
		t.Fatalf(
			"expected manifest name %q, got %q",
			manifest.Name,
			got.Name,
		)
	}

	if len(got.Modules) != 2 {
		t.Fatalf("expected 2 resolved modules, got %d", len(got.Modules))
	}

	if got.Modules[0] != manifest.Modules[0] {
		t.Fatalf(
			"expected first module %q@%q, got %q@%q",
			manifest.Modules[0].Name,
			manifest.Modules[0].Version,
			got.Modules[0].Name,
			got.Modules[0].Version,
		)
	}

	if got.Modules[1] != manifest.Modules[1] {
		t.Fatalf(
			"expected second module %q@%q, got %q@%q",
			manifest.Modules[1].Name,
			manifest.Modules[1].Version,
			got.Modules[1].Name,
			got.Modules[1].Version,
		)
	}
}

func TestResolveUsesExactVersionMatch(t *testing.T) {
	manifest := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v2"},
		},
	}

	available := []Module{
		{Name: "http", Version: "v1"},
		{Name: "http", Version: "v2"},
	}

	got, err := Resolve(manifest, available)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Modules) != 1 {
		t.Fatalf("expected 1 resolved module, got %d", len(got.Modules))
	}

	if got.Modules[0].Version != "v2" {
		t.Fatalf(
			"expected exact version v2, got %q",
			got.Modules[0].Version,
		)
	}
}

func TestResolvePreservesManifestOrder(t *testing.T) {
	manifest := Manifest{
		Version: "v1",
		Name:    "ordered",
		Modules: []Module{
			{Name: "third", Version: "v1"},
			{Name: "first", Version: "v1"},
			{Name: "second", Version: "v2"},
		},
	}

	available := []Module{
		{Name: "second", Version: "v2"},
		{Name: "first", Version: "v1"},
		{Name: "third", Version: "v1"},
	}

	got, err := Resolve(manifest, available)
	if err != nil {
		t.Fatal(err)
	}

	want := []Module{
		{Name: "third", Version: "v1"},
		{Name: "first", Version: "v1"},
		{Name: "second", Version: "v2"},
	}

	if len(got.Modules) != len(want) {
		t.Fatalf(
			"expected %d modules, got %d",
			len(want),
			len(got.Modules),
		)
	}

	for i := range want {
		if got.Modules[i] != want[i] {
			t.Fatalf(
				"module %d: expected %q@%q, got %q@%q",
				i,
				want[i].Name,
				want[i].Version,
				got.Modules[i].Name,
				got.Modules[i].Version,
			)
		}
	}
}

func TestResolveMissingModule(t *testing.T) {
	manifest := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v2"},
		},
	}

	available := []Module{
		{Name: "http", Version: "v1"},
	}

	_, err := Resolve(manifest, available)
	if err == nil {
		t.Fatal("expected resolution error")
	}

	forgeErr, ok := err.(*forgeerrors.Error)
	if !ok {
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

func TestResolveRejectsInvalidManifest(t *testing.T) {
	manifest := Manifest{
		Version: "",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v1"},
		},
	}

	_, err := Resolve(manifest, []Module{
		{Name: "http", Version: "v1"},
	})

	if err == nil {
		t.Fatal("expected validation error")
	}

	forgeErr, ok := err.(*forgeerrors.Error)
	if !ok {
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

func TestResolveAllowsEmptyModules(t *testing.T) {
	manifest := Manifest{
		Version: "v1",
		Name:    "empty",
		Modules: nil,
	}

	got, err := Resolve(manifest, []Module{
		{Name: "http", Version: "v1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Modules) != 0 {
		t.Fatalf(
			"expected 0 resolved modules, got %d",
			len(got.Modules),
		)
	}
}

func TestResolveDuplicateAvailableModulesIsDeterministic(t *testing.T) {
	manifest := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{Name: "http", Version: "v1"},
		},
	}

	available := []Module{
		{Name: "http", Version: "v1"},
		{Name: "http", Version: "v1"},
	}

	got, err := Resolve(manifest, available)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Modules) != 1 {
		t.Fatalf(
			"expected 1 resolved module, got %d",
			len(got.Modules),
		)
	}

	if got.Modules[0].Name != "http" ||
		got.Modules[0].Version != "v1" {
		t.Fatalf(
			"unexpected resolved module: %q@%q",
			got.Modules[0].Name,
			got.Modules[0].Version,
		)
	}
}
