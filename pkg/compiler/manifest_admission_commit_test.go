package compiler

import (
	"errors"
	"reflect"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

func seededManifestAdmissionRegistries(
	t *testing.T,
) (*registry.Registry, *PackageSourceRegistry) {
	t.Helper()

	packages := registry.New()
	sources := NewPackageSourceRegistry()

	if err := packages.EnsureAll([]registry.Package{
		{Name: "existing", Version: "v1"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := sources.EnsureAll([]PackageSource{
		{
			Name:       "existing",
			Version:    "v1",
			ImportPath: "example.com/forge/existing",
		},
	}); err != nil {
		t.Fatal(err)
	}

	return packages, sources
}

func requireManifestAdmissionRegistryState(
	t *testing.T,
	packages *registry.Registry,
	sources *PackageSourceRegistry,
	wantPackages []registry.Package,
	wantSources []PackageSource,
) {
	t.Helper()

	if got := packages.List(); !reflect.DeepEqual(got, wantPackages) {
		t.Fatalf("expected packages %#v, got %#v", wantPackages, got)
	}

	if got := sources.List(); !reflect.DeepEqual(got, wantSources) {
		t.Fatalf("expected sources %#v, got %#v", wantSources, got)
	}
}

func manifestAdmissionCommitConflictManifest() manifest.Manifest {
	return manifest.Manifest{
		Name:    "conflict",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "a",
				Version:    "v1",
				ImportPath: "example.com/forge/a",
			},
			{
				Name:       "b",
				Version:    "v1",
				ImportPath: "example.com/forge/requested-b",
			},
			{
				Name:       "c",
				Version:    "v1",
				ImportPath: "example.com/forge/c",
			},
		},
	}
}

func TestAdmitManifest(t *testing.T) {
	packages := registry.New()
	sources := NewPackageSourceRegistry()

	admission, err := AdmitManifest(
		manifestAdmissionTestManifest(),
		packages,
		sources,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := packages.List(), manifestAdmissionTestPackages(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected packages %#v, got %#v", want, got)
	}

	if got, want := sources.List(), manifestAdmissionTestSources(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sources %#v, got %#v", want, got)
	}

	if got, want := admission.Sources(), manifestAdmissionTestSources(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected candidates %#v, got %#v", want, got)
	}

	wantOrder := []string{"cache@v1", "logger@v1", "web@v1"}
	plan := admission.BuildPlan()
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

func TestAdmitManifestPreservesApplicationEntrypointAndCanonicalSource(
	t *testing.T,
) {
	packages := registry.New()
	sources := NewPackageSourceRegistry()
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Entrypoint: &manifest.ApplicationEntrypoint{
			Module:  "app",
			Version: "v1",
		},
		Modules: []manifest.Module{
			{
				Name:       "app",
				Version:    "v1",
				ImportPath: " example.com/demo/app ",
			},
		},
	}

	admission, err := AdmitManifest(m, packages, sources)
	if err != nil {
		t.Fatal(err)
	}

	wantEntrypoint := manifest.ApplicationEntrypoint{
		Module:  "app",
		Version: "v1",
	}
	gotEntrypoint, ok := admission.ApplicationEntrypoint()
	if !ok || gotEntrypoint != wantEntrypoint {
		t.Fatalf(
			"expected entrypoint %#v, true; got %#v, %t",
			wantEntrypoint,
			gotEntrypoint,
			ok,
		)
	}

	wantSource := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "example.com/demo/app",
	}
	gotSource, err := sources.Resolve(
		gotEntrypoint.Module,
		gotEntrypoint.Version,
	)
	if err != nil {
		t.Fatal(err)
	}

	if gotSource != wantSource {
		t.Fatalf("expected canonical source %#v, got %#v", wantSource, gotSource)
	}

	if got := admission.Sources(); !reflect.DeepEqual(
		got,
		[]PackageSource{wantSource},
	) {
		t.Fatalf("expected normalized sources %#v, got %#v", []PackageSource{wantSource}, got)
	}
}

func TestAdmitManifestApplicationEntrypointIsIdempotent(t *testing.T) {
	packages := registry.New()
	sources := NewPackageSourceRegistry()
	m := manifestAdmissionTestManifest()
	m.Entrypoint = &manifest.ApplicationEntrypoint{
		Module:  "web",
		Version: "v1",
	}

	first, err := AdmitManifest(m, packages, sources)
	if err != nil {
		t.Fatal(err)
	}

	wantEntrypoint, wantHasEntrypoint := first.ApplicationEntrypoint()
	wantPlan := first.BuildPlan()
	wantPackages := first.Packages()
	wantSources := first.Sources()
	wantRegistryPackages := packages.List()
	wantRegistrySources := sources.List()

	for i := 0; i < 2; i++ {
		admission, admitErr := AdmitManifest(m, packages, sources)
		if admitErr != nil {
			t.Fatal(admitErr)
		}

		gotEntrypoint, gotHasEntrypoint := admission.ApplicationEntrypoint()
		if gotEntrypoint != wantEntrypoint ||
			gotHasEntrypoint != wantHasEntrypoint ||
			!reflect.DeepEqual(admission.BuildPlan(), wantPlan) ||
			!reflect.DeepEqual(admission.Packages(), wantPackages) ||
			!reflect.DeepEqual(admission.Sources(), wantSources) {
			t.Fatalf("admission %d differs from first admission", i+2)
		}
	}

	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		wantRegistryPackages,
		wantRegistrySources,
	)

	if packages.Count() != len(wantRegistryPackages) ||
		sources.Count() != len(wantRegistrySources) {
		t.Fatalf(
			"expected stable counts %d/%d, got %d/%d",
			len(wantRegistryPackages),
			len(wantRegistrySources),
			packages.Count(),
			sources.Count(),
		)
	}
}

func TestAdmitManifestIsIdempotent(t *testing.T) {
	packages := registry.New()
	sources := NewPackageSourceRegistry()
	m := manifestAdmissionTestManifest()

	first, err := AdmitManifest(m, packages, sources)
	if err != nil {
		t.Fatal(err)
	}

	wantPackages := packages.List()
	wantSources := sources.List()
	wantPlan := first.BuildPlan()

	for i := 0; i < 2; i++ {
		admission, err := AdmitManifest(m, packages, sources)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(admission.BuildPlan(), wantPlan) ||
			!reflect.DeepEqual(admission.Packages(), first.Packages()) ||
			!reflect.DeepEqual(admission.Sources(), first.Sources()) {
			t.Fatalf("admission %d differs from first admission", i+2)
		}
	}

	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		wantPackages,
		wantSources,
	)

	if packages.Count() != len(wantPackages) ||
		sources.Count() != len(wantSources) {
		t.Fatalf(
			"expected stable counts %d/%d, got %d/%d",
			len(wantPackages),
			len(wantSources),
			packages.Count(),
			sources.Count(),
		)
	}
}

func TestAdmitManifestRejectsNilPackageRegistry(t *testing.T) {
	sources := NewPackageSourceRegistry()
	if err := sources.EnsureAll([]PackageSource{
		{
			Name:       "existing",
			Version:    "v1",
			ImportPath: "example.com/forge/existing",
		},
	}); err != nil {
		t.Fatal(err)
	}

	before := sources.List()
	admission, err := AdmitManifest(
		manifestAdmissionTestManifest(),
		nil,
		sources,
	)

	if !errors.Is(err, registry.ErrInvalidPackage) {
		t.Fatalf("expected ErrInvalidPackage, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
	if got := sources.List(); !reflect.DeepEqual(got, before) {
		t.Fatalf("expected sources %#v, got %#v", before, got)
	}
}

func TestAdmitManifestRejectsNilSourceRegistry(t *testing.T) {
	packages := registry.New()
	if err := packages.EnsureAll([]registry.Package{
		{Name: "existing", Version: "v1"},
	}); err != nil {
		t.Fatal(err)
	}

	before := packages.List()
	admission, err := AdmitManifest(
		manifestAdmissionTestManifest(),
		packages,
		nil,
	)

	if !errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
	if got := packages.List(); !reflect.DeepEqual(got, before) {
		t.Fatalf("expected packages %#v, got %#v", before, got)
	}
}

func TestAdmitManifestRejectsInvalidManifestWithoutMutation(t *testing.T) {
	tests := []struct {
		name string
		m    manifest.Manifest
	}{
		{
			name: "missing name",
			m:    manifest.Manifest{Version: "v1"},
		},
		{
			name: "missing module version",
			m: manifest.Manifest{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages, sources := seededManifestAdmissionRegistries(t)
			beforePackages := packages.List()
			beforeSources := sources.List()

			admission, err := AdmitManifest(tt.m, packages, sources)
			requireManifestAdmissionForgeErrorCode(
				t,
				err,
				forgeerrors.CodeInvalidManifest,
			)
			requireZeroManifestAdmissionPlan(t, admission)
			requireManifestAdmissionRegistryState(
				t,
				packages,
				sources,
				beforePackages,
				beforeSources,
			)
		})
	}
}

func TestAdmitManifestRejectsMissingImportPathWithoutMutation(
	t *testing.T,
) {
	packages, sources := seededManifestAdmissionRegistries(t)
	beforePackages := packages.List()
	beforeSources := sources.List()
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Modules: []manifest.Module{
			{Name: "app", Version: "v1", ImportPath: "   "},
		},
	}

	admission, err := AdmitManifest(m, packages, sources)
	if !errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		beforePackages,
		beforeSources,
	)
}

func TestAdmitManifestRejectsExistingSourceConflictWithoutPackageMutation(
	t *testing.T,
) {
	packages := registry.New()
	sources := NewPackageSourceRegistry()
	canonicalPackage := registry.Package{Name: "b", Version: "v1"}
	canonicalSource := PackageSource{
		Name:       "b",
		Version:    "v1",
		ImportPath: "example.com/forge/canonical-b",
	}

	if err := packages.EnsureAll([]registry.Package{canonicalPackage}); err != nil {
		t.Fatal(err)
	}

	if err := sources.EnsureAll([]PackageSource{canonicalSource}); err != nil {
		t.Fatal(err)
	}

	beforePackages := packages.List()
	beforeSources := sources.List()
	admission, err := AdmitManifest(
		manifestAdmissionCommitConflictManifest(),
		packages,
		sources,
	)

	if !errors.Is(err, ErrPackageSourceConflict) {
		t.Fatalf("expected ErrPackageSourceConflict, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		beforePackages,
		beforeSources,
	)

	for _, name := range []string{"a", "c"} {
		if _, getErr := packages.Get(name, "v1"); !errors.Is(
			getErr,
			registry.ErrPackageNotFound,
		) {
			t.Fatalf("expected package %s@v1 to be absent, got %v", name, getErr)
		}

		if _, resolveErr := sources.Resolve(name, "v1"); !errors.Is(
			resolveErr,
			ErrPackageSourceNotFound,
		) {
			t.Fatalf("expected source %s@v1 to be absent, got %v", name, resolveErr)
		}
	}

	got, resolveErr := sources.Resolve("b", "v1")
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}

	if !reflect.DeepEqual(got, canonicalSource) {
		t.Fatalf("expected canonical source %#v, got %#v", canonicalSource, got)
	}
}

func TestAdmitManifestApplicationEntrypointSourceConflictReturnsZeroPlan(
	t *testing.T,
) {
	packages := registry.New()
	sources := NewPackageSourceRegistry()
	canonicalPackage := registry.Package{Name: "app", Version: "v1"}
	canonicalSource := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "example.com/demo/canonical-app",
	}

	if err := packages.EnsureAll([]registry.Package{canonicalPackage}); err != nil {
		t.Fatal(err)
	}
	if err := sources.EnsureAll([]PackageSource{canonicalSource}); err != nil {
		t.Fatal(err)
	}

	beforePackages := packages.List()
	beforeSources := sources.List()
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Entrypoint: &manifest.ApplicationEntrypoint{
			Module:  "app",
			Version: "v1",
		},
		Modules: []manifest.Module{
			{
				Name:       "app",
				Version:    "v1",
				ImportPath: "example.com/demo/requested-app",
			},
		},
	}

	admission, err := AdmitManifest(m, packages, sources)
	if !errors.Is(err, ErrPackageSourceConflict) {
		t.Fatalf("expected ErrPackageSourceConflict, got %v", err)
	}

	requireZeroManifestAdmissionPlan(t, admission)
	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		beforePackages,
		beforeSources,
	)
}

func TestAdmitManifestRejectsMissingDependencyWithoutMutation(t *testing.T) {
	packages, sources := seededManifestAdmissionRegistries(t)
	beforePackages := packages.List()
	beforeSources := sources.List()
	m := manifest.Manifest{
		Name:    "missing",
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
		},
	}

	admission, err := AdmitManifest(m, packages, sources)
	requireManifestAdmissionForgeErrorCode(t, err, forgeerrors.CodeNotFound)
	requireZeroManifestAdmissionPlan(t, admission)
	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		beforePackages,
		beforeSources,
	)
}

func TestAdmitManifestRejectsCycleWithoutMutation(t *testing.T) {
	packages, sources := seededManifestAdmissionRegistries(t)
	beforePackages := packages.List()
	beforeSources := sources.List()
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

	admission, err := AdmitManifest(m, packages, sources)
	requireManifestAdmissionForgeErrorCode(
		t,
		err,
		forgeerrors.CodeInvalidManifest,
	)
	requireZeroManifestAdmissionPlan(t, admission)
	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		beforePackages,
		beforeSources,
	)
}

func TestAdmitManifestSupportsEmptyModules(t *testing.T) {
	packages, sources := seededManifestAdmissionRegistries(t)
	beforePackages := packages.List()
	beforeSources := sources.List()
	m := manifest.Manifest{Name: "empty", Version: "v1"}

	admission, err := AdmitManifest(m, packages, sources)
	if err != nil {
		t.Fatal(err)
	}

	if len(admission.Packages()) != 0 || len(admission.Sources()) != 0 {
		t.Fatalf("expected empty candidates, got %#v/%#v", admission.Packages(), admission.Sources())
	}

	plan := admission.BuildPlan()
	if plan.ManifestName != m.Name || plan.ManifestVersion != m.Version ||
		len(plan.Steps) != 0 {
		t.Fatalf("unexpected empty build plan: %#v", plan)
	}

	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		beforePackages,
		beforeSources,
	)
}

func TestAdmitManifestPreservesExistingRegistrationOrder(t *testing.T) {
	packages, sources := seededManifestAdmissionRegistries(t)

	if _, err := AdmitManifest(
		manifestAdmissionTestManifest(),
		packages,
		sources,
	); err != nil {
		t.Fatal(err)
	}

	wantPackages := append(
		[]registry.Package{{Name: "existing", Version: "v1"}},
		manifestAdmissionTestPackages()...,
	)
	wantSources := append(
		[]PackageSource{
			{
				Name:       "existing",
				Version:    "v1",
				ImportPath: "example.com/forge/existing",
			},
		},
		manifestAdmissionTestSources()...,
	)

	requireManifestAdmissionRegistryState(
		t,
		packages,
		sources,
		wantPackages,
		wantSources,
	)
}

func TestAdmitManifestDoesNotMutateManifest(t *testing.T) {
	m := manifestAdmissionTestManifest()
	original := cloneManifestAdmissionTestManifest(m)

	if _, err := AdmitManifest(
		m,
		registry.New(),
		NewPackageSourceRegistry(),
	); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m, original) {
		t.Fatalf("expected manifest %#v, got %#v", original, m)
	}
}

func TestCommitManifestAdmissionRevalidatesLiveSourceConflict(
	t *testing.T,
) {
	m := manifestAdmissionCommitConflictManifest()
	admission, err := PrepareManifestAdmission(m, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	packages := registry.New()
	sources := NewPackageSourceRegistry()
	canonical := PackageSource{
		Name:       "b",
		Version:    "v1",
		ImportPath: "example.com/forge/canonical-b",
	}
	if err := sources.Ensure(canonical); err != nil {
		t.Fatal(err)
	}

	err = commitManifestAdmission(admission, packages, sources)
	if !errors.Is(err, ErrPackageSourceConflict) {
		t.Fatalf("expected ErrPackageSourceConflict, got %v", err)
	}

	if packages.Count() != 0 {
		t.Fatalf("expected no packages, got %#v", packages.List())
	}

	wantSources := []PackageSource{canonical}
	if got := sources.List(); !reflect.DeepEqual(got, wantSources) {
		t.Fatalf("expected sources %#v, got %#v", wantSources, got)
	}

	for _, name := range []string{"a", "c"} {
		if _, resolveErr := sources.Resolve(name, "v1"); !errors.Is(
			resolveErr,
			ErrPackageSourceNotFound,
		) {
			t.Fatalf("expected source %s@v1 to be absent, got %v", name, resolveErr)
		}
	}
}

func TestCommitManifestAdmissionDoesNotMutateAdmissionPlan(t *testing.T) {
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

	if err := commitManifestAdmission(
		admission,
		registry.New(),
		NewPackageSourceRegistry(),
	); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(admission.BuildPlan(), wantPlan) ||
		!reflect.DeepEqual(admission.Packages(), wantPackages) ||
		!reflect.DeepEqual(admission.Sources(), wantSources) {
		t.Fatal("commit mutated admission plan")
	}
}
