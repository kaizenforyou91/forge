package compiler

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

func manifestAdmissionTestManifest() manifest.Manifest {
	return manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "web",
				Version:    "v1",
				ImportPath: " example.com/forge/web ",
				Dependencies: []manifest.Dependency{
					{Name: "cache", Version: "v1"},
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:       "logger",
				Version:    "v1",
				ImportPath: " example.com/forge/logger ",
			},
			{
				Name:       "cache",
				Version:    "v1",
				ImportPath: " example.com/forge/cache ",
			},
		},
	}
}

func manifestAdmissionTestPackages() []registry.Package {
	return []registry.Package{
		{Name: "web", Version: "v1"},
		{Name: "logger", Version: "v1"},
		{Name: "cache", Version: "v1"},
	}
}

func manifestAdmissionTestSources() []PackageSource {
	return []PackageSource{
		{
			Name:       "web",
			Version:    "v1",
			ImportPath: "example.com/forge/web",
		},
		{
			Name:       "logger",
			Version:    "v1",
			ImportPath: "example.com/forge/logger",
		},
		{
			Name:       "cache",
			Version:    "v1",
			ImportPath: "example.com/forge/cache",
		},
	}
}

func cloneManifestAdmissionTestManifest(
	m manifest.Manifest,
) manifest.Manifest {
	clone := m
	if m.Entrypoint != nil {
		entrypoint := *m.Entrypoint
		clone.Entrypoint = &entrypoint
	}

	if m.Modules == nil {
		return clone
	}

	clone.Modules = make([]manifest.Module, len(m.Modules))
	for i, module := range m.Modules {
		clone.Modules[i] = module
		if module.Dependencies == nil {
			continue
		}

		clone.Modules[i].Dependencies = make(
			[]manifest.Dependency,
			len(module.Dependencies),
		)
		copy(clone.Modules[i].Dependencies, module.Dependencies)
	}

	return clone
}

func requireZeroManifestAdmissionPlan(
	t *testing.T,
	plan ManifestAdmissionPlan,
) {
	t.Helper()

	if !reflect.DeepEqual(plan, ManifestAdmissionPlan{}) {
		t.Fatalf("expected zero admission plan, got %#v", plan)
	}
}

func requireManifestAdmissionForgeErrorCode(
	t *testing.T,
	err error,
	want forgeerrors.Code,
) {
	t.Helper()

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}

	if forgeErr.Code != want {
		t.Fatalf("expected error code %s, got %s", want, forgeErr.Code)
	}
}

func TestPrepareManifestAdmission(t *testing.T) {
	admission, err := PrepareManifestAdmission(
		manifestAdmissionTestManifest(),
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := admission.Packages(), manifestAdmissionTestPackages(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected packages %#v, got %#v", want, got)
	}

	if got, want := admission.Sources(), manifestAdmissionTestSources(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sources %#v, got %#v", want, got)
	}

	plan := admission.BuildPlan()
	if plan.ManifestName != "demo" || plan.ManifestVersion != "v1" {
		t.Fatalf(
			"expected manifest identity demo@v1, got %s@%s",
			plan.ManifestName,
			plan.ManifestVersion,
		)
	}

	wantOrder := []string{"cache@v1", "logger@v1", "web@v1"}
	if len(plan.Steps) != len(wantOrder) {
		t.Fatalf("expected %d steps, got %d", len(wantOrder), len(plan.Steps))
	}

	for i, want := range wantOrder {
		if plan.Steps[i].Module != want {
			t.Fatalf(
				"step %d: expected %q, got %q",
				i,
				want,
				plan.Steps[i].Module,
			)
		}
	}
}

func TestPrepareManifestAdmissionAcceptsEquivalentStrictLoaderResults(t *testing.T) {
	directory := t.TempDir()
	yamlPath := filepath.Join(directory, "forge.yaml")
	jsonPath := filepath.Join(directory, "forge.json")
	yamlData := []byte(`version: v1
name: aplikasi
entrypoint: {module: app, version: v1}
modules:
  - name: app
    version: v1
    import_path: " example.com/app "
`)
	jsonData := []byte(`{"version":"v1","name":"aplikasi","entrypoint":{"module":"app","version":"v1"},"modules":[{"name":"app","version":"v1","import_path":" example.com/app "}]}`)
	if err := os.WriteFile(yamlPath, yamlData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0o644); err != nil {
		t.Fatal(err)
	}

	yamlManifest, err := manifest.LoadYAML(yamlPath)
	if err != nil {
		t.Fatal(err)
	}
	jsonManifest, err := manifest.LoadJSON(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	yamlOriginal := cloneManifestAdmissionTestManifest(yamlManifest)
	jsonOriginal := cloneManifestAdmissionTestManifest(jsonManifest)

	yamlAdmission, err := PrepareManifestAdmission(yamlManifest, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	jsonAdmission, err := PrepareManifestAdmission(jsonManifest, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(yamlAdmission.BuildPlan(), jsonAdmission.BuildPlan()) ||
		!reflect.DeepEqual(yamlAdmission.Packages(), jsonAdmission.Packages()) ||
		!reflect.DeepEqual(yamlAdmission.Sources(), jsonAdmission.Sources()) {
		t.Fatalf("strict loader admissions differ:\nYAML: %#v\nJSON: %#v", yamlAdmission, jsonAdmission)
	}
	if !reflect.DeepEqual(yamlManifest, yamlOriginal) {
		t.Fatalf("YAML manifest mutated: want %#v, got %#v", yamlOriginal, yamlManifest)
	}
	if !reflect.DeepEqual(jsonManifest, jsonOriginal) {
		t.Fatalf("JSON manifest mutated: want %#v, got %#v", jsonOriginal, jsonManifest)
	}
}

func TestPrepareManifestAdmissionPreservesOptionalApplicationEntrypoint(
	t *testing.T,
) {
	t.Run("absent", func(t *testing.T) {
		admission, err := PrepareManifestAdmission(
			manifestAdmissionTestManifest(),
			nil,
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}

		entrypoint, ok := admission.ApplicationEntrypoint()
		if ok || entrypoint != (manifest.ApplicationEntrypoint{}) {
			t.Fatalf(
				"expected absent entrypoint, got %#v, %t",
				entrypoint,
				ok,
			)
		}
	})

	t.Run("present", func(t *testing.T) {
		m := manifestAdmissionTestManifest()
		want := manifest.ApplicationEntrypoint{
			Module:  "web",
			Version: "v1",
		}
		m.Entrypoint = &want

		admission, err := PrepareManifestAdmission(m, nil, nil)
		if err != nil {
			t.Fatal(err)
		}

		entrypoint, ok := admission.ApplicationEntrypoint()
		if !ok || entrypoint != want {
			t.Fatalf(
				"expected entrypoint %#v, true; got %#v, %t",
				want,
				entrypoint,
				ok,
			)
		}
	})
}

func TestPrepareManifestAdmissionSnapshotsApplicationEntrypoint(
	t *testing.T,
) {
	m := manifestAdmissionTestManifest()
	m.Entrypoint = &manifest.ApplicationEntrypoint{
		Module:  "web",
		Version: "v1",
	}

	admission, err := PrepareManifestAdmission(m, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	m.Entrypoint.Module = "mutated"
	m.Entrypoint.Version = "v9"
	m.Entrypoint = &manifest.ApplicationEntrypoint{
		Module:  "replacement",
		Version: "v2",
	}

	want := manifest.ApplicationEntrypoint{Module: "web", Version: "v1"}
	got, ok := admission.ApplicationEntrypoint()
	if !ok || got != want {
		t.Fatalf(
			"expected entrypoint %#v, true; got %#v, %t",
			want,
			got,
			ok,
		)
	}
}

func TestManifestAdmissionApplicationEntrypointAccessorReturnsCopy(
	t *testing.T,
) {
	m := manifestAdmissionTestManifest()
	m.Entrypoint = &manifest.ApplicationEntrypoint{
		Module:  "web",
		Version: "v1",
	}

	admission, err := PrepareManifestAdmission(m, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	entrypoint, ok := admission.ApplicationEntrypoint()
	if !ok {
		t.Fatal("expected application entrypoint")
	}

	entrypoint.Module = "mutated"
	entrypoint.Version = "v9"

	want := manifest.ApplicationEntrypoint{Module: "web", Version: "v1"}
	got, ok := admission.ApplicationEntrypoint()
	if !ok || got != want {
		t.Fatalf(
			"expected entrypoint %#v, true; got %#v, %t",
			want,
			got,
			ok,
		)
	}
}

func TestPrepareManifestAdmissionEntrypointDoesNotAffectPreparedEvidence(
	t *testing.T,
) {
	withoutEntrypoint := manifestAdmissionTestManifest()
	withEntrypoint := cloneManifestAdmissionTestManifest(withoutEntrypoint)
	withEntrypoint.Entrypoint = &manifest.ApplicationEntrypoint{
		Module:  "web",
		Version: "v1",
	}

	without, err := PrepareManifestAdmission(withoutEntrypoint, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	with, err := PrepareManifestAdmission(withEntrypoint, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(with.BuildPlan(), without.BuildPlan()) {
		t.Fatalf(
			"entrypoint changed build plan: without %#v, with %#v",
			without.BuildPlan(),
			with.BuildPlan(),
		)
	}

	if !reflect.DeepEqual(with.Packages(), without.Packages()) {
		t.Fatalf(
			"entrypoint changed packages: without %#v, with %#v",
			without.Packages(),
			with.Packages(),
		)
	}

	if !reflect.DeepEqual(with.Sources(), without.Sources()) {
		t.Fatalf(
			"entrypoint changed sources: without %#v, with %#v",
			without.Sources(),
			with.Sources(),
		)
	}
}

func TestPrepareManifestAdmissionEntrypointFailureReturnsZeroPlan(
	t *testing.T,
) {
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Entrypoint: &manifest.ApplicationEntrypoint{
			Module:  "app",
			Version: "v1",
		},
		Modules: []manifest.Module{
			{Name: "app", Version: "v1", ImportPath: "   "},
		},
	}

	admission, err := PrepareManifestAdmission(m, nil, nil)
	if !errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
}

func TestPrepareManifestAdmissionPreservesDependencyMetadata(
	t *testing.T,
) {
	admission, err := PrepareManifestAdmission(
		manifestAdmissionTestManifest(),
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	plan := admission.BuildPlan()
	var webStep manifest.BuildStep

	for _, step := range plan.Steps {
		if step.Module == "web@v1" {
			webStep = step
			break
		}
	}

	want := []string{"cache@v1", "logger@v1"}
	if !reflect.DeepEqual(webStep.Dependencies, want) {
		t.Fatalf(
			"expected dependencies %#v, got %#v",
			want,
			webStep.Dependencies,
		)
	}
}

func TestPrepareManifestAdmissionAcceptsExistingIdenticalSource(
	t *testing.T,
) {
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "app",
				Version:    "v1",
				ImportPath: " example.com/forge/app ",
			},
		},
	}
	existing := []PackageSource{
		{
			Name:       " app ",
			Version:    " v1 ",
			ImportPath: "example.com/forge/app",
		},
	}

	admission, err := PrepareManifestAdmission(m, nil, existing)
	if err != nil {
		t.Fatal(err)
	}

	want := []PackageSource{
		{
			Name:       "app",
			Version:    "v1",
			ImportPath: "example.com/forge/app",
		},
	}
	if got := admission.Sources(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sources %#v, got %#v", want, got)
	}
}

func TestPrepareManifestAdmissionRejectsExistingSourceConflict(
	t *testing.T,
) {
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "app",
				Version:    "v1",
				ImportPath: "example.com/forge/app",
			},
		},
	}
	existing := []PackageSource{
		{
			Name:       "app",
			Version:    "v1",
			ImportPath: "example.com/forge/conflict",
		},
	}

	admission, err := PrepareManifestAdmission(m, nil, existing)
	if !errors.Is(err, ErrPackageSourceConflict) {
		t.Fatalf("expected ErrPackageSourceConflict, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
}

func TestPrepareManifestAdmissionRejectsMissingImportPath(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
	}{
		{name: "empty", importPath: ""},
		{name: "whitespace", importPath: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := manifest.Manifest{
				Name:    "demo",
				Version: "v1",
				Modules: []manifest.Module{
					{
						Name:       "app",
						Version:    "v1",
						ImportPath: tt.importPath,
					},
				},
			}

			admission, err := PrepareManifestAdmission(m, nil, nil)
			if !errors.Is(err, ErrInvalidPackageSource) {
				t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
			}

			if !strings.Contains(err.Error(), "app") {
				t.Fatalf("expected module context in error, got %v", err)
			}

			requireZeroManifestAdmissionPlan(t, admission)
		})
	}
}

func TestPrepareManifestAdmissionRejectsInvalidManifest(t *testing.T) {
	validModule := manifest.Module{
		Name:       "app",
		Version:    "v1",
		ImportPath: "example.com/forge/app",
	}

	tests := []struct {
		name     string
		manifest manifest.Manifest
	}{
		{
			name: "missing manifest name",
			manifest: manifest.Manifest{
				Version: "v1",
			},
		},
		{
			name: "missing manifest version",
			manifest: manifest.Manifest{
				Name: "demo",
			},
		},
		{
			name: "missing module name",
			manifest: manifest.Manifest{
				Name:    "demo",
				Version: "v1",
				Modules: []manifest.Module{
					{
						Version:    "v1",
						ImportPath: "example.com/forge/app",
					},
				},
			},
		},
		{
			name: "missing module version",
			manifest: manifest.Manifest{
				Name:    "demo",
				Version: "v1",
				Modules: []manifest.Module{
					{
						Name:       "app",
						ImportPath: "example.com/forge/app",
					},
				},
			},
		},
		{
			name: "duplicate module",
			manifest: manifest.Manifest{
				Name:    "demo",
				Version: "v1",
				Modules: []manifest.Module{
					validModule,
					validModule,
				},
			},
		},
		{
			name: "ambiguous manifest identity",
			manifest: manifest.Manifest{
				Name:    "demo@other",
				Version: "v1",
				Modules: []manifest.Module{validModule},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			admission, err := PrepareManifestAdmission(
				tt.manifest,
				nil,
				nil,
			)

			requireManifestAdmissionForgeErrorCode(
				t,
				err,
				forgeerrors.CodeInvalidManifest,
			)
			requireZeroManifestAdmissionPlan(t, admission)
		})
	}
}

func TestPrepareManifestAdmissionRejectsMissingDependency(t *testing.T) {
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "web",
				Version:    "v1",
				ImportPath: "example.com/forge/web",
				Dependencies: []manifest.Dependency{
					{Name: "http", Version: "v1"},
				},
			},
		},
	}

	admission, err := PrepareManifestAdmission(m, nil, nil)
	requireManifestAdmissionForgeErrorCode(
		t,
		err,
		forgeerrors.CodeNotFound,
	)
	requireZeroManifestAdmissionPlan(t, admission)
}

func TestPrepareManifestAdmissionRejectsDependencyCycle(t *testing.T) {
	m := manifest.Manifest{
		Name:    "cycle",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "a",
				Version:    "v1",
				ImportPath: "example.com/forge/a",
				Dependencies: []manifest.Dependency{
					{Name: "b", Version: "v1"},
				},
			},
			{
				Name:       "b",
				Version:    "v1",
				ImportPath: "example.com/forge/b",
				Dependencies: []manifest.Dependency{
					{Name: "a", Version: "v1"},
				},
			},
		},
	}

	admission, err := PrepareManifestAdmission(m, nil, nil)
	requireManifestAdmissionForgeErrorCode(
		t,
		err,
		forgeerrors.CodeInvalidManifest,
	)
	requireZeroManifestAdmissionPlan(t, admission)
}

func TestPrepareManifestAdmissionRejectsInvalidExistingPackageSnapshot(
	t *testing.T,
) {
	m := manifest.Manifest{Name: "empty", Version: "v1"}
	existing := []registry.Package{
		{Name: "existing"},
	}

	admission, err := PrepareManifestAdmission(m, existing, nil)
	if !errors.Is(err, registry.ErrInvalidPackage) {
		t.Fatalf("expected ErrInvalidPackage, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
}

func TestPrepareManifestAdmissionRejectsInvalidExistingSourceSnapshot(
	t *testing.T,
) {
	m := manifest.Manifest{Name: "empty", Version: "v1"}
	existing := []PackageSource{
		{
			Name:       "existing",
			Version:    "v1",
			ImportPath: "   ",
		},
	}

	admission, err := PrepareManifestAdmission(m, nil, existing)
	if !errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
}

func TestPrepareManifestAdmissionReturnsManifestCandidatesOnly(
	t *testing.T,
) {
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "app",
				Version:    "v1",
				ImportPath: "example.com/forge/app",
			},
		},
	}
	existingPackages := []registry.Package{
		{Name: "existing", Version: "v1"},
	}
	existingSources := []PackageSource{
		{
			Name:       "existing",
			Version:    "v1",
			ImportPath: "example.com/forge/existing",
		},
	}

	admission, err := PrepareManifestAdmission(
		m,
		existingPackages,
		existingSources,
	)
	if err != nil {
		t.Fatal(err)
	}

	wantPackages := []registry.Package{
		{Name: "app", Version: "v1"},
	}
	if got := admission.Packages(); !reflect.DeepEqual(got, wantPackages) {
		t.Fatalf("expected packages %#v, got %#v", wantPackages, got)
	}

	wantSources := []PackageSource{
		{
			Name:       "app",
			Version:    "v1",
			ImportPath: "example.com/forge/app",
		},
	}
	if got := admission.Sources(); !reflect.DeepEqual(got, wantSources) {
		t.Fatalf("expected sources %#v, got %#v", wantSources, got)
	}
}

func TestPrepareManifestAdmissionPreservesDeclarationOrderForAdmission(
	t *testing.T,
) {
	m := manifest.Manifest{
		Name:    "ordered",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "zeta",
				Version:    "v1",
				ImportPath: "example.com/forge/zeta",
			},
			{
				Name:       "alpha",
				Version:    "v1",
				ImportPath: "example.com/forge/alpha",
			},
			{
				Name:       "middle",
				Version:    "v1",
				ImportPath: "example.com/forge/middle",
			},
		},
	}

	admission, err := PrepareManifestAdmission(m, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	wantNames := []string{"zeta", "alpha", "middle"}
	packages := admission.Packages()
	sources := admission.Sources()

	if len(packages) != len(wantNames) || len(sources) != len(wantNames) {
		t.Fatalf(
			"expected %d candidates, got %d packages and %d sources",
			len(wantNames),
			len(packages),
			len(sources),
		)
	}

	for i, want := range wantNames {
		if packages[i].Name != want {
			t.Fatalf(
				"package %d: expected %q, got %q",
				i,
				want,
				packages[i].Name,
			)
		}

		if sources[i].Name != want {
			t.Fatalf(
				"source %d: expected %q, got %q",
				i,
				want,
				sources[i].Name,
			)
		}
	}
}

func TestPrepareManifestAdmissionSupportsEmptyModules(t *testing.T) {
	m := manifest.Manifest{Name: "empty", Version: "v1"}

	admission, err := PrepareManifestAdmission(m, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(admission.Packages()) != 0 {
		t.Fatalf("expected no package candidates, got %#v", admission.Packages())
	}

	if len(admission.Sources()) != 0 {
		t.Fatalf("expected no source candidates, got %#v", admission.Sources())
	}

	plan := admission.BuildPlan()
	if plan.ManifestName != m.Name || plan.ManifestVersion != m.Version {
		t.Fatalf(
			"expected manifest identity %s@%s, got %s@%s",
			m.Name,
			m.Version,
			plan.ManifestName,
			plan.ManifestVersion,
		)
	}

	if len(plan.Steps) != 0 {
		t.Fatalf("expected no build steps, got %#v", plan.Steps)
	}

	entrypoint, ok := admission.ApplicationEntrypoint()
	if ok || entrypoint != (manifest.ApplicationEntrypoint{}) {
		t.Fatalf(
			"expected absent entrypoint, got %#v, %t",
			entrypoint,
			ok,
		)
	}
}

func TestPrepareManifestAdmissionDoesNotMutateInputs(t *testing.T) {
	m := manifestAdmissionTestManifest()
	originalManifest := cloneManifestAdmissionTestManifest(m)
	existingPackages := []registry.Package{
		{Name: "existing", Version: "v1"},
	}
	originalPackages := append(
		[]registry.Package(nil),
		existingPackages...,
	)
	existingSources := []PackageSource{
		{
			Name:       "existing",
			Version:    "v1",
			ImportPath: "example.com/forge/existing",
		},
	}
	originalSources := append(
		[]PackageSource(nil),
		existingSources...,
	)

	if _, err := PrepareManifestAdmission(
		m,
		existingPackages,
		existingSources,
	); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m, originalManifest) {
		t.Fatalf("manifest mutated: expected %#v, got %#v", originalManifest, m)
	}

	if !reflect.DeepEqual(existingPackages, originalPackages) {
		t.Fatalf(
			"package snapshot mutated: expected %#v, got %#v",
			originalPackages,
			existingPackages,
		)
	}

	if !reflect.DeepEqual(existingSources, originalSources) {
		t.Fatalf(
			"source snapshot mutated: expected %#v, got %#v",
			originalSources,
			existingSources,
		)
	}
}

func TestPrepareManifestAdmissionIsDeterministic(t *testing.T) {
	m := manifestAdmissionTestManifest()
	existingPackages := []registry.Package{
		{Name: "existing", Version: "v1"},
	}
	existingSources := []PackageSource{
		{
			Name:       "existing",
			Version:    "v1",
			ImportPath: "example.com/forge/existing",
		},
	}

	first, err := PrepareManifestAdmission(
		m,
		existingPackages,
		existingSources,
	)
	if err != nil {
		t.Fatal(err)
	}

	second, err := PrepareManifestAdmission(
		m,
		existingPackages,
		existingSources,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first.BuildPlan(), second.BuildPlan()) {
		t.Fatalf(
			"build plans differ: first %#v, second %#v",
			first.BuildPlan(),
			second.BuildPlan(),
		)
	}

	if !reflect.DeepEqual(first.Packages(), second.Packages()) {
		t.Fatalf(
			"package candidates differ: first %#v, second %#v",
			first.Packages(),
			second.Packages(),
		)
	}

	if !reflect.DeepEqual(first.Sources(), second.Sources()) {
		t.Fatalf(
			"source candidates differ: first %#v, second %#v",
			first.Sources(),
			second.Sources(),
		)
	}
}

func TestManifestAdmissionPlanAccessorsReturnIndependentCopies(
	t *testing.T,
) {
	admission, err := PrepareManifestAdmission(
		manifestAdmissionTestManifest(),
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	wantPlan := admission.BuildPlan()
	wantPackages := admission.Packages()
	wantSources := admission.Sources()

	mutatedPlan := admission.BuildPlan()
	mutatedPlan.ManifestName = "mutated"
	mutatedPlan.Steps[0].Module = "mutated@v9"
	mutatedPlan.Steps[len(mutatedPlan.Steps)-1].Dependencies[0] =
		"mutated@v9"

	mutatedPackages := admission.Packages()
	mutatedPackages[0].Name = "mutated"

	mutatedSources := admission.Sources()
	mutatedSources[0].ImportPath = "example.com/mutated"

	if got := admission.BuildPlan(); !reflect.DeepEqual(got, wantPlan) {
		t.Fatalf("expected build plan %#v, got %#v", wantPlan, got)
	}

	if got := admission.Packages(); !reflect.DeepEqual(got, wantPackages) {
		t.Fatalf("expected packages %#v, got %#v", wantPackages, got)
	}

	if got := admission.Sources(); !reflect.DeepEqual(got, wantSources) {
		t.Fatalf("expected sources %#v, got %#v", wantSources, got)
	}
}

func TestManifestAdmissionPlanZeroValueAccessors(t *testing.T) {
	var admission ManifestAdmissionPlan

	plan := admission.BuildPlan()
	if !reflect.DeepEqual(plan, manifest.BuildPlan{}) {
		t.Fatalf("expected zero build plan, got %#v", plan)
	}

	plan.ManifestName = "mutated"
	plan.Steps = append(plan.Steps, manifest.BuildStep{Module: "mutated@v9"})

	packages := admission.Packages()
	packages = append(packages, registry.Package{Name: "mutated", Version: "v9"})

	sources := admission.Sources()
	sources = append(sources, PackageSource{
		Name:       "mutated",
		Version:    "v9",
		ImportPath: "example.com/mutated",
	})

	entrypoint, hasEntrypoint := admission.ApplicationEntrypoint()
	entrypoint.Module = "mutated"
	entrypoint.Version = "v9"
	if hasEntrypoint {
		t.Fatal("expected zero-value plan to have no application entrypoint")
	}

	if got := admission.BuildPlan(); !reflect.DeepEqual(
		got,
		manifest.BuildPlan{},
	) {
		t.Fatalf("expected zero build plan, got %#v", got)
	}

	if got := admission.Packages(); len(got) != 0 {
		t.Fatalf("expected no packages, got %#v", got)
	}

	if got := admission.Sources(); len(got) != 0 {
		t.Fatalf("expected no sources, got %#v", got)
	}

	gotEntrypoint, ok := admission.ApplicationEntrypoint()
	if ok || gotEntrypoint != (manifest.ApplicationEntrypoint{}) {
		t.Fatalf(
			"expected absent entrypoint, got %#v, %t",
			gotEntrypoint,
			ok,
		)
	}
}
