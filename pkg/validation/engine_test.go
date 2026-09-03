package validation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

func validManifest() manifest.Manifest {
	return manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []manifest.Module{
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}
}

func TestNewEngine(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	if engine == nil {
		t.Fatal("expected engine")
	}

	if engine.Count() != 0 {
		t.Fatalf("expected 0 validators, got %d", engine.Count())
	}
}

func TestEngineAdd(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	err = engine.Add(ValidatorFunc(func(manifest.Manifest) error {
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	if engine.Count() != 1 {
		t.Fatalf("expected 1 validator, got %d", engine.Count())
	}
}

func TestEngineRejectsNilValidator(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Add(nil); !errors.Is(err, ErrNilValidator) {
		t.Fatalf("expected ErrNilValidator, got %v", err)
	}
}

func TestNewEngineRejectsNilValidator(t *testing.T) {
	_, err := NewEngine(nil)

	if !errors.Is(err, ErrNilValidator) {
		t.Fatalf("expected ErrNilValidator, got %v", err)
	}
}

func TestEngineValidatesManifest(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Validate(validManifest()); err != nil {
		t.Fatalf("expected valid manifest, got %v", err)
	}
}

func TestEngineRunsValidatorsInOrder(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	var order []int

	if err := engine.Add(ValidatorFunc(func(manifest.Manifest) error {
		order = append(order, 1)
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	if err := engine.Add(ValidatorFunc(func(manifest.Manifest) error {
		order = append(order, 2)
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	if err := engine.Add(ValidatorFunc(func(manifest.Manifest) error {
		order = append(order, 3)
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	if err := engine.Validate(validManifest()); err != nil {
		t.Fatal(err)
	}

	want := []int{1, 2, 3}

	if len(order) != len(want) {
		t.Fatalf("expected order length %d, got %d", len(want), len(order))
	}

	for i := range want {
		if order[i] != want[i] {
			t.Fatalf(
				"validator %d: expected %d, got %d",
				i,
				want[i],
				order[i],
			)
		}
	}
}

func TestEngineStopsAtFirstError(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	expectedErr := errors.New("semantic validation failed")
	secondExecuted := false

	if err := engine.Add(ValidatorFunc(func(manifest.Manifest) error {
		return expectedErr
	})); err != nil {
		t.Fatal(err)
	}

	if err := engine.Add(ValidatorFunc(func(manifest.Manifest) error {
		secondExecuted = true
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	err = engine.Validate(validManifest())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected original validation error, got %v", err)
	}

	if secondExecuted {
		t.Fatal("validator after first error should not execute")
	}
}

func TestEngineRunsStructuralValidationFirst(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	executed := false

	if err := engine.Add(ValidatorFunc(func(manifest.Manifest) error {
		executed = true
		return nil
	})); err != nil {
		t.Fatal(err)
	}

	invalid := manifest.Manifest{
		Version: "",
		Name:    "demo",
	}

	if err := engine.Validate(invalid); err == nil {
		t.Fatal("expected structural validation error")
	}

	if executed {
		t.Fatal("semantic validator should not run after structural validation failure")
	}
}

func TestEnginePropagatesValidatorError(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	expectedErr := errors.New("validation failed")

	if err := engine.Add(ValidatorFunc(func(manifest.Manifest) error {
		return expectedErr
	})); err != nil {
		t.Fatal(err)
	}

	err = engine.Validate(validManifest())

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected propagated error, got %v", err)
	}
}

func TestEngineEmptyValidators(t *testing.T) {
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Validate(validManifest()); err != nil {
		t.Fatalf("expected valid manifest, got %v", err)
	}
}

func TestEngineIsolatesCallerAndValidatorsFromManifestMutation(t *testing.T) {
	original := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Entrypoint: &manifest.ApplicationEntrypoint{
			Module:  "app",
			Version: "v1",
		},
		Modules: []manifest.Module{
			{
				Name:    "app",
				Version: "v1",
				Dependencies: []manifest.Dependency{
					{Name: "dep", Version: "v1"},
				},
			},
			{Name: "dep", Version: "v1"},
		},
	}
	want := cloneEngineTestManifest(original)
	secondSawOriginal := false

	engine, err := NewEngine(
		ValidatorFunc(func(received manifest.Manifest) error {
			received.Entrypoint.Module = "mutated-entrypoint"
			received.Modules[0].Name = "mutated-module"
			received.Modules[0].Dependencies[0].Name = "mutated-dependency"
			received.Modules[0].Dependencies = append(
				received.Modules[0].Dependencies,
				manifest.Dependency{Name: "extra", Version: "v1"},
			)
			received.Modules = append(
				received.Modules,
				manifest.Module{Name: "extra", Version: "v1"},
			)
			return nil
		}),
		ValidatorFunc(func(received manifest.Manifest) error {
			secondSawOriginal = reflect.DeepEqual(received, want)
			return nil
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Validate(original); err != nil {
		t.Fatal(err)
	}
	if !secondSawOriginal {
		t.Fatal("second validator observed mutations from first validator")
	}
	if !reflect.DeepEqual(original, want) {
		t.Fatalf("caller manifest mutated:\nwant: %#v\ngot:  %#v", want, original)
	}
}

func cloneEngineTestManifest(original manifest.Manifest) manifest.Manifest {
	clone := original
	if original.Entrypoint != nil {
		entrypoint := *original.Entrypoint
		clone.Entrypoint = &entrypoint
	}
	if original.Modules != nil {
		clone.Modules = make([]manifest.Module, len(original.Modules))
		for i, module := range original.Modules {
			clone.Modules[i] = module
			clone.Modules[i].Dependencies = append(
				[]manifest.Dependency(nil),
				module.Dependencies...,
			)
		}
	}
	return clone
}
