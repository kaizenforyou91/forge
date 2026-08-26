package compiler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

func admitRunnableManifestForTest(
	t *testing.T,
	m manifest.Manifest,
) (ManifestAdmissionPlan, *PackageSourceRegistry) {
	t.Helper()

	sources := NewPackageSourceRegistry()
	admission, err := AdmitManifest(m, registry.New(), sources)
	if err != nil {
		t.Fatal(err)
	}

	return admission, sources
}

func prepareRunnableManifestForTest(
	t *testing.T,
	m manifest.Manifest,
) ManifestAdmissionPlan {
	t.Helper()

	admission, err := PrepareManifestAdmission(m, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	return admission
}

func newFakeRunnableManifestCompiler(
	t *testing.T,
) (*RunnableManifestCompiler, *fakeRunnableExecutableBuilder, *fakeRunnablePackagePackager) {
	t.Helper()

	builder := &fakeRunnableExecutableBuilder{}
	packager := &fakeRunnablePackagePackager{}
	compiler, err := newRunnableManifestCompiler(builder, packager)
	if err != nil {
		t.Fatal(err)
	}

	return compiler, builder, packager
}

func runnableManifestApplication(
	importPath string,
) manifest.Manifest {
	return manifest.Manifest{
		Name:    "runnable-manifest",
		Version: "v1",
		Entrypoint: &manifest.ApplicationEntrypoint{
			Module:  "app",
			Version: "v1",
		},
		Modules: []manifest.Module{
			{
				Name:       "app",
				Version:    "v1",
				ImportPath: importPath,
			},
		},
	}
}

func malformedRunnableManifestAdmission(
	sources []PackageSource,
) ManifestAdmissionPlan {
	return ManifestAdmissionPlan{
		buildPlan: manifest.BuildPlan{
			ManifestName:    "runnable-manifest",
			ManifestVersion: "v1",
			Steps: []manifest.BuildStep{
				{Module: "app@v1"},
			},
		},
		sources: sources,
		applicationEntrypoint: manifest.ApplicationEntrypoint{
			Module:  "app",
			Version: "v1",
		},
		hasApplicationEntrypoint: true,
	}
}

func compileFakeRunnableManifest(
	t *testing.T,
	compiler *RunnableManifestCompiler,
	admission ManifestAdmissionPlan,
) error {
	t.Helper()

	return compiler.Compile(context.Background(), RunnableManifestRequest{
		Admission:        admission,
		WorkingDirectory: t.TempDir(),
		OutputPath:       filepath.Join(t.TempDir(), "runnable-v2.zip"),
	})
}

func TestRunnableManifestCompilerRequiresAdmittedEntrypoint(t *testing.T) {
	m := manifest.Manifest{
		Name:    "library",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:       "library",
				Version:    "v1",
				ImportPath: "example.com/library",
			},
		},
	}
	admission, _ := admitRunnableManifestForTest(t, m)
	coordinator, builder, packager := newFakeRunnableManifestCompiler(t)

	for _, test := range []struct {
		name      string
		admission ManifestAdmissionPlan
	}{
		{name: "admitted manifest without entrypoint", admission: admission},
		{name: "zero admission plan", admission: ManifestAdmissionPlan{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputPath := filepath.Join(t.TempDir(), "must-not-exist.zip")
			err := coordinator.Compile(context.Background(), RunnableManifestRequest{
				Admission:        test.admission,
				WorkingDirectory: t.TempDir(),
				OutputPath:       outputPath,
			})
			if !errors.Is(err, ErrInvalidApplicationEntrypoint) {
				t.Fatalf("expected ErrInvalidApplicationEntrypoint, got %v", err)
			}
			if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
				t.Fatalf("expected no output package, got %v", statErr)
			}
		})
	}

	if len(builder.requests) != 0 || len(packager.bundles) != 0 {
		t.Fatal("missing entrypoint must fail before build and package")
	}
}

func TestRunnableManifestCompilerRejectsMissingAdmittedSource(t *testing.T) {
	coordinator, builder, packager := newFakeRunnableManifestCompiler(t)
	err := compileFakeRunnableManifest(
		t,
		coordinator,
		malformedRunnableManifestAdmission(nil),
	)
	if !errors.Is(err, ErrInvalidApplicationEntrypoint) ||
		!errors.Is(err, ErrPackageSourceNotFound) {
		t.Fatalf("expected missing admitted source classifications, got %v", err)
	}
	if len(builder.requests) != 0 || len(packager.bundles) != 0 {
		t.Fatal("missing source must fail before build and package")
	}
}

func TestRunnableManifestCompilerRejectsDuplicateAdmittedSource(t *testing.T) {
	coordinator, builder, packager := newFakeRunnableManifestCompiler(t)
	source := PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: "example.com/source-a",
	}
	err := compileFakeRunnableManifest(
		t,
		coordinator,
		malformedRunnableManifestAdmission([]PackageSource{source, source}),
	)
	if !errors.Is(err, ErrInvalidApplicationEntrypoint) ||
		!errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf("expected duplicate admitted source classifications, got %v", err)
	}
	if len(builder.requests) != 0 || len(packager.bundles) != 0 {
		t.Fatal("duplicate source must fail before build and package")
	}
}

func TestRunnableManifestCompilerRejectsNoncanonicalAdmittedSource(t *testing.T) {
	for _, test := range []struct {
		name   string
		source PackageSource
	}{
		{
			name: "surrounding import path whitespace",
			source: PackageSource{
				Name:       "app",
				Version:    "v1",
				ImportPath: " example.com/app ",
			},
		},
		{
			name: "blank import path",
			source: PackageSource{
				Name:       "app",
				Version:    "v1",
				ImportPath: " ",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			coordinator, builder, packager := newFakeRunnableManifestCompiler(t)
			err := compileFakeRunnableManifest(
				t,
				coordinator,
				malformedRunnableManifestAdmission([]PackageSource{test.source}),
			)
			if !errors.Is(err, ErrInvalidApplicationEntrypoint) ||
				!errors.Is(err, ErrInvalidPackageSource) {
				t.Fatalf("expected noncanonical admitted source classifications, got %v", err)
			}
			if len(builder.requests) != 0 || len(packager.bundles) != 0 {
				t.Fatal("noncanonical source must fail before build and package")
			}
		})
	}
}

func TestRunnableManifestCompilerBindsAdmittedSourceAgainstDivergentResolver(
	t *testing.T,
) {
	const (
		sourceA = "example.com/source-a"
		sourceB = "example.com/source-b"
	)
	admission := prepareRunnableManifestForTest(
		t,
		runnableManifestApplication(" "+sourceA+" "),
	)

	divergentRegistry := NewPackageSourceRegistry()
	if err := divergentRegistry.Ensure(PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: sourceB,
	}); err != nil {
		t.Fatal(err)
	}
	divergent, err := divergentRegistry.Resolve("app", "v1")
	if err != nil || divergent.ImportPath != sourceB {
		t.Fatalf("divergent resolver fixture is invalid: %#v, %v", divergent, err)
	}

	coordinator, builder, packager := newFakeRunnableManifestCompiler(t)
	if err := compileFakeRunnableManifest(t, coordinator, admission); err != nil {
		t.Fatal(err)
	}

	if len(builder.requests) != 1 {
		t.Fatalf("expected one executable build, got %d", len(builder.requests))
	}
	if got := builder.requests[0].ImportPath; got != sourceA {
		t.Fatalf("expected admitted builder import path %q, got %q", sourceA, got)
	}
	if got := builder.requests[0].Entrypoint; got != (RuntimeEntrypoint{Module: "app", Version: "v1"}) {
		t.Fatalf("unexpected builder entrypoint %#v", got)
	}
	if len(packager.bundles) != 1 || len(packager.bundles[0].Artifacts) != 1 {
		t.Fatalf("unexpected packaged bundles %#v", packager.bundles)
	}
	bundle := packager.bundles[0]
	if got := bundle.Artifacts[0].ImportPath; got != sourceA {
		t.Fatalf("expected admitted artifact import path %q, got %q", sourceA, got)
	}
	if bundle.Runtime == nil ||
		bundle.Runtime.Entrypoint != (RuntimeEntrypoint{Module: "app", Version: "v1"}) {
		t.Fatalf("unexpected runtime entrypoint %#v", bundle.Runtime)
	}
	if bundle.Artifacts[0].ImportPath == sourceB {
		t.Fatal("divergent resolver source affected manifest-driven output")
	}
}

func TestRunnableManifestCompilerUsesPreparedAdmissionAsSourceAuthority(t *testing.T) {
	const sourceA = "example.com/prepared-app"
	admission := prepareRunnableManifestForTest(
		t,
		runnableManifestApplication(" "+sourceA+" "),
	)
	coordinator, builder, packager := newFakeRunnableManifestCompiler(t)

	if err := compileFakeRunnableManifest(t, coordinator, admission); err != nil {
		t.Fatal(err)
	}
	if len(builder.requests) != 1 || builder.requests[0].ImportPath != sourceA {
		t.Fatalf("prepared plan did not supply source authority: %#v", builder.requests)
	}
	if len(packager.bundles) != 1 ||
		packager.bundles[0].Artifacts[0].ImportPath != sourceA {
		t.Fatalf("prepared source not preserved in artifact: %#v", packager.bundles)
	}
}

func TestRunnableManifestCompilerUsesImmutableCommittedAdmissionEvidence(
	t *testing.T,
) {
	const (
		sourceA = "example.com/source-a"
		sourceB = "example.com/source-b"
	)
	m := runnableManifestApplication(" " + sourceA + " ")
	admission, liveRegistry := admitRunnableManifestForTest(t, m)

	conflictErr := liveRegistry.Ensure(PackageSource{
		Name:       "app",
		Version:    "v1",
		ImportPath: sourceB,
	})
	if !errors.Is(conflictErr, ErrPackageSourceConflict) {
		t.Fatalf("expected live registry conflict, got %v", conflictErr)
	}
	if err := liveRegistry.Ensure(PackageSource{
		Name:       "unrelated",
		Version:    "v1",
		ImportPath: "example.com/unrelated",
	}); err != nil {
		t.Fatal(err)
	}

	m.Entrypoint.Module = "mutated-manifest"
	m.Entrypoint.Version = "v9"
	m.Modules[0].ImportPath = sourceB
	entrypointCopy, ok := admission.ApplicationEntrypoint()
	if !ok {
		t.Fatal("expected admitted entrypoint")
	}
	entrypointCopy.Module = "mutated-copy"
	entrypointCopy.Version = "v8"
	sourcesCopy := admission.Sources()
	sourcesCopy[0].ImportPath = sourceB

	coordinator, builder, packager := newFakeRunnableManifestCompiler(t)
	if err := compileFakeRunnableManifest(t, coordinator, admission); err != nil {
		t.Fatal(err)
	}
	if len(builder.requests) != 1 || builder.requests[0].ImportPath != sourceA {
		t.Fatalf("mutated caller state changed builder authority: %#v", builder.requests)
	}
	if builder.requests[0].Entrypoint != (RuntimeEntrypoint{Module: "app", Version: "v1"}) {
		t.Fatalf("mutated caller state changed entrypoint: %#v", builder.requests[0])
	}
	if len(packager.bundles) != 1 ||
		packager.bundles[0].Artifacts[0].ImportPath != sourceA {
		t.Fatalf("mutated caller state changed artifact authority: %#v", packager.bundles)
	}
}

func TestRunnableManifestCompilerConvertsExactAdmittedEntrypointWithoutInference(
	t *testing.T,
) {
	m := manifest.Manifest{
		Name:    "topology",
		Version: "v1",
		Entrypoint: &manifest.ApplicationEntrypoint{
			Module:  "middle",
			Version: "v2",
		},
		Modules: []manifest.Module{
			{
				Name:       "root",
				Version:    "v3",
				ImportPath: "example.com/root",
				Dependencies: []manifest.Dependency{
					{Name: "middle", Version: "v2"},
				},
			},
			{
				Name:       "middle",
				Version:    "v2",
				ImportPath: "example.com/middle",
				Dependencies: []manifest.Dependency{
					{Name: "leaf", Version: "v1"},
				},
			},
			{
				Name:       "leaf",
				Version:    "v1",
				ImportPath: "example.com/leaf",
			},
		},
	}
	admission, _ := admitRunnableManifestForTest(t, m)
	plan := admission.BuildPlan()
	if len(plan.Steps) != 3 ||
		plan.Steps[0].Module == "middle@v2" ||
		plan.Steps[len(plan.Steps)-1].Module == "middle@v2" {
		t.Fatalf("fixture must place entrypoint between plan boundaries: %#v", plan.Steps)
	}

	coordinator, builder, packager := newFakeRunnableManifestCompiler(t)
	ctx := context.WithValue(context.Background(), struct{}{}, "preserved")
	request := RunnableManifestRequest{
		Admission:        admission,
		WorkingDirectory: t.TempDir(),
		OutputPath:       filepath.Join(t.TempDir(), "topology-v2.zip"),
	}
	if err := coordinator.Compile(ctx, request); err != nil {
		t.Fatal(err)
	}

	if len(builder.requests) != 1 || len(builder.contexts) != 1 {
		t.Fatalf("expected one build delegation, got %d", len(builder.requests))
	}
	wantEntrypoint := RuntimeEntrypoint{Module: "middle", Version: "v2"}
	got := builder.requests[0]
	if got.Entrypoint != wantEntrypoint || got.ImportPath != "example.com/middle" {
		t.Fatalf("unexpected builder request %#v", got)
	}
	if builder.contexts[0] != ctx {
		t.Fatal("context was not delegated")
	}
	if len(packager.bundles) != 1 ||
		packager.bundles[0].Runtime == nil ||
		packager.bundles[0].Runtime.Entrypoint != wantEntrypoint {
		t.Fatalf("unexpected packaged runtime %#v", packager.bundles)
	}
}

func TestRunnableManifestCompilerPreservesRunnablePackageFilesystemErrors(
	t *testing.T,
) {
	admission, _ := admitRunnableManifestForTest(
		t,
		runnableManifestApplication("example.com/app"),
	)
	coordinator, builder, _ := newFakeRunnableManifestCompiler(t)

	t.Run("working directory", func(t *testing.T) {
		err := coordinator.Compile(context.Background(), RunnableManifestRequest{
			Admission:        admission,
			WorkingDirectory: " ",
			OutputPath:       filepath.Join(t.TempDir(), "output.zip"),
		})
		if err == nil {
			t.Fatal("expected working-directory validation error")
		}
	})

	t.Run("output path", func(t *testing.T) {
		err := coordinator.Compile(context.Background(), RunnableManifestRequest{
			Admission:        admission,
			WorkingDirectory: t.TempDir(),
			OutputPath:       " ",
		})
		if !errors.Is(err, ErrInvalidArtifactPackage) {
			t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
		}
	})

	if len(builder.requests) != 0 {
		t.Fatal("invalid filesystem requests must fail before executable build")
	}
}

func TestRunnableManifestCompilerPreservesNonMainSourceFailure(t *testing.T) {
	t.Setenv("GOTELEMETRY", "off")
	workingDirectory, _ := runnablePackageRepositoryPaths(t)
	admission, _ := admitRunnableManifestForTest(
		t,
		runnableManifestApplication(testNonMainImportPath),
	)
	builder, err := NewGoApplicationExecutableBuilder(NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewRunnableManifestCompiler(builder, NewZIPPackager())
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "non-main-v2.zip")

	err = coordinator.Compile(context.Background(), RunnableManifestRequest{
		Admission:        admission,
		WorkingDirectory: workingDirectory,
		OutputPath:       outputPath,
	})
	if !errors.Is(err, ErrInvalidApplicationEntrypoint) {
		t.Fatalf("expected ErrInvalidApplicationEntrypoint, got %v", err)
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no package output, got %v", statErr)
	}
}

func TestNewRunnableManifestCompilerRejectsIncompleteCoordinator(t *testing.T) {
	builder, err := NewGoApplicationExecutableBuilder(&fakeCommandRunner{})
	if err != nil {
		t.Fatal(err)
	}

	coordinator, err := NewRunnableManifestCompiler(nil, NewZIPPackager())
	if !errors.Is(err, ErrExecutableBuildFailed) || coordinator != nil {
		t.Fatalf("expected nil builder failure, got %#v, %v", coordinator, err)
	}
	coordinator, err = NewRunnableManifestCompiler(builder, nil)
	if !errors.Is(err, ErrInvalidArtifactPackage) || coordinator != nil {
		t.Fatalf("expected nil packager failure, got %#v, %v", coordinator, err)
	}

	var nilCoordinator *RunnableManifestCompiler
	err = nilCoordinator.Compile(context.Background(), RunnableManifestRequest{})
	if !errors.Is(err, ErrExecutableBuildFailed) {
		t.Fatalf("expected incomplete coordinator failure, got %v", err)
	}

	incomplete := &RunnableManifestCompiler{}
	err = incomplete.Compile(context.Background(), RunnableManifestRequest{})
	if !errors.Is(err, ErrExecutableBuildFailed) {
		t.Fatalf("expected incomplete coordinator failure, got %v", err)
	}
	incomplete = &RunnableManifestCompiler{builder: &fakeRunnableExecutableBuilder{}}
	err = incomplete.Compile(context.Background(), RunnableManifestRequest{})
	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf("expected incomplete packager failure, got %v", err)
	}
}

func TestRunnableManifestCompilerCreatesStrictlyVerifiedSignedPackageV2(
	t *testing.T,
) {
	t.Setenv("GOTELEMETRY", "off")
	workingDirectory, fixturePath := runnablePackageRepositoryPaths(t)
	fixtureBefore := snapshotRunnableApplicationFixture(t, fixturePath)
	m := runnableManifestApplication(" " + testRunnableApplicationImportPath + " ")
	admittedEntrypoint := *m.Entrypoint
	admission, sources := admitRunnableManifestForTest(t, m)

	m.Entrypoint.Module = "mutated-after-admission"
	m.Entrypoint.Version = "v9"
	m.Modules[0].ImportPath = "example.com/source-b"
	accessorCopy, ok := admission.ApplicationEntrypoint()
	if !ok {
		t.Fatal("expected admitted entrypoint")
	}
	accessorCopy.Module = "mutated-accessor-copy"
	accessorCopy.Version = "v8"
	sourceCopies := admission.Sources()
	sourceCopies[0].ImportPath = "example.com/source-b"

	canonicalSource, err := sources.Resolve(
		admittedEntrypoint.Module,
		admittedEntrypoint.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	if canonicalSource.ImportPath != testRunnableApplicationImportPath {
		t.Fatalf("expected canonical import path %q, got %q", testRunnableApplicationImportPath, canonicalSource.ImportPath)
	}
	divergentRegistry := NewPackageSourceRegistry()
	if err := divergentRegistry.Ensure(PackageSource{
		Name:       admittedEntrypoint.Module,
		Version:    admittedEntrypoint.Version,
		ImportPath: "example.com/source-b",
	}); err != nil {
		t.Fatal(err)
	}

	signer, verifier := trustedTestSignerAndVerifier(t)
	builder, err := NewGoApplicationExecutableBuilder(NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewRunnableManifestCompiler(
		builder,
		NewZIPPackagerWithSigner(signer),
	)
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(t.TempDir(), "admitted-runnable-v2.zip")
	if err := coordinator.Compile(context.Background(), RunnableManifestRequest{
		Admission:        admission,
		WorkingDirectory: workingDirectory,
		OutputPath:       outputPath,
	}); err != nil {
		t.Fatal(err)
	}

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reader.ReadDetailed(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.PackageFormatVersion != 2 || result.BundleSchemaVersion != 2 {
		t.Fatalf("expected version pair 2/2, got %d/%d", result.PackageFormatVersion, result.BundleSchemaVersion)
	}
	if result.VerifiedSignerKeyID != "forge-dev" {
		t.Fatalf("expected verified signer forge-dev, got %q", result.VerifiedSignerKeyID)
	}

	wantEntrypoint := RuntimeEntrypoint{
		Module:  admittedEntrypoint.Module,
		Version: admittedEntrypoint.Version,
	}
	bundle := result.Bundle
	if bundle.Runtime == nil ||
		bundle.Runtime.Kind != RuntimeKindApplicationExecutable ||
		bundle.Runtime.Entrypoint != wantEntrypoint ||
		bundle.Runtime.TargetOS != runtime.GOOS ||
		bundle.Runtime.TargetArch != runtime.GOARCH {
		t.Fatalf("unexpected runtime descriptor %#v", bundle.Runtime)
	}
	if len(bundle.Artifacts) != 1 {
		t.Fatalf("expected one artifact, got %d", len(bundle.Artifacts))
	}
	wantArtifact := Artifact{
		Module:     admittedEntrypoint.Module,
		Version:    admittedEntrypoint.Version,
		ImportPath: canonicalSource.ImportPath,
	}
	if bundle.Artifacts[0] != wantArtifact {
		t.Fatalf("expected artifact %#v, got %#v", wantArtifact, bundle.Artifacts[0])
	}
	payload := result.Payloads[admittedEntrypoint.Module+"@"+admittedEntrypoint.Version]
	if len(result.Payloads) != 1 || len(payload) == 0 {
		t.Fatalf("expected one non-empty executable payload, got %#v", result.Payloads)
	}

	fixtureAfter := snapshotRunnableApplicationFixture(t, fixturePath)
	if !reflect.DeepEqual(fixtureAfter, fixtureBefore) {
		t.Fatalf("real build mutated source fixture\nbefore %#v\nafter  %#v", fixtureBefore, fixtureAfter)
	}
}
