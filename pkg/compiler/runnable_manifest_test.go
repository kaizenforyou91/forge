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

type recordingRunnableManifestPackageCompiler struct {
	contexts []context.Context
	requests []RunnablePackageRequest
	err      error
}

func (c *recordingRunnableManifestPackageCompiler) Compile(
	ctx context.Context,
	request RunnablePackageRequest,
) error {
	c.contexts = append(c.contexts, ctx)
	c.requests = append(c.requests, request)
	return c.err
}

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
	recorder := &recordingRunnableManifestPackageCompiler{}
	coordinator := &RunnableManifestCompiler{compiler: recorder}

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

	if len(recorder.requests) != 0 {
		t.Fatalf("missing entrypoint must not delegate, got %#v", recorder.requests)
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

	recorder := &recordingRunnableManifestPackageCompiler{}
	coordinator := &RunnableManifestCompiler{compiler: recorder}
	ctx := context.WithValue(context.Background(), struct{}{}, "preserved")
	request := RunnableManifestRequest{
		Admission:        admission,
		WorkingDirectory: t.TempDir(),
		OutputPath:       filepath.Join(t.TempDir(), "topology-v2.zip"),
	}
	if err := coordinator.Compile(ctx, request); err != nil {
		t.Fatal(err)
	}

	if len(recorder.requests) != 1 || len(recorder.contexts) != 1 {
		t.Fatalf("expected one delegation, got %d", len(recorder.requests))
	}
	wantEntrypoint := RuntimeEntrypoint{Module: "middle", Version: "v2"}
	got := recorder.requests[0]
	if got.Entrypoint != wantEntrypoint {
		t.Fatalf("expected entrypoint %#v, got %#v", wantEntrypoint, got.Entrypoint)
	}
	if !reflect.DeepEqual(got.Plan, admission.BuildPlan()) ||
		got.WorkingDirectory != request.WorkingDirectory ||
		got.OutputPath != request.OutputPath ||
		recorder.contexts[0] != ctx {
		t.Fatalf("unexpected delegated request %#v", got)
	}
}

func TestRunnableManifestCompilerUsesImmutableAdmissionEntrypoint(t *testing.T) {
	m := runnableManifestApplication("example.com/app")
	admission, _ := admitRunnableManifestForTest(t, m)

	m.Entrypoint.Module = "mutated-manifest"
	m.Entrypoint.Version = "v9"
	entrypointCopy, ok := admission.ApplicationEntrypoint()
	if !ok {
		t.Fatal("expected admitted entrypoint")
	}
	entrypointCopy.Module = "mutated-copy"
	entrypointCopy.Version = "v8"

	recorder := &recordingRunnableManifestPackageCompiler{}
	coordinator := &RunnableManifestCompiler{compiler: recorder}
	if err := coordinator.Compile(context.Background(), RunnableManifestRequest{
		Admission:        admission,
		WorkingDirectory: t.TempDir(),
		OutputPath:       filepath.Join(t.TempDir(), "immutable-v2.zip"),
	}); err != nil {
		t.Fatal(err)
	}

	want := RuntimeEntrypoint{Module: "app", Version: "v1"}
	if got := recorder.requests[0].Entrypoint; got != want {
		t.Fatalf("expected admitted entrypoint %#v, got %#v", want, got)
	}
}

func TestRunnableManifestCompilerDelegatesFilesystemValidation(t *testing.T) {
	admission, _ := admitRunnableManifestForTest(
		t,
		runnableManifestApplication("example.com/app"),
	)
	wantErr := errors.New("delegated runnable package validation")

	for _, test := range []struct {
		name             string
		workingDirectory string
		outputPath       string
	}{
		{
			name:             "working directory",
			workingDirectory: " ",
			outputPath:       filepath.Join(t.TempDir(), "output.zip"),
		},
		{
			name:             "output path",
			workingDirectory: t.TempDir(),
			outputPath:       " ",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := &recordingRunnableManifestPackageCompiler{err: wantErr}
			coordinator := &RunnableManifestCompiler{compiler: recorder}
			err := coordinator.Compile(context.Background(), RunnableManifestRequest{
				Admission:        admission,
				WorkingDirectory: test.workingDirectory,
				OutputPath:       test.outputPath,
			})
			if !errors.Is(err, wantErr) {
				t.Fatalf("expected delegated error, got %v", err)
			}
			if len(recorder.requests) != 1 ||
				recorder.requests[0].WorkingDirectory != test.workingDirectory ||
				recorder.requests[0].OutputPath != test.outputPath {
				t.Fatalf("filesystem values were not delegated exactly: %#v", recorder.requests)
			}
		})
	}
}

func TestRunnableManifestCompilerPreservesRunnablePackageFilesystemErrors(
	t *testing.T,
) {
	admission, sources := admitRunnableManifestForTest(
		t,
		runnableManifestApplication("example.com/app"),
	)
	builder := &fakeRunnableExecutableBuilder{}
	runnableCompiler, err := newRunnablePackageCompiler(
		sources,
		builder,
		&fakeRunnablePackagePackager{},
	)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewRunnableManifestCompiler(runnableCompiler)
	if err != nil {
		t.Fatal(err)
	}

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

func TestRunnableManifestCompilerPreservesSourceResolutionFailure(t *testing.T) {
	m := runnableManifestApplication("example.com/app")
	admission, err := PrepareManifestAdmission(m, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	builder := &fakeRunnableExecutableBuilder{}
	runnableCompiler, err := newRunnablePackageCompiler(
		NewPackageSourceRegistry(),
		builder,
		&fakeRunnablePackagePackager{},
	)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewRunnableManifestCompiler(runnableCompiler)
	if err != nil {
		t.Fatal(err)
	}

	err = coordinator.Compile(context.Background(), RunnableManifestRequest{
		Admission:        admission,
		WorkingDirectory: t.TempDir(),
		OutputPath:       filepath.Join(t.TempDir(), "missing-source-v2.zip"),
	})
	if !errors.Is(err, ErrInvalidApplicationEntrypoint) ||
		!errors.Is(err, ErrPackageSourceNotFound) {
		t.Fatalf("expected preserved entrypoint source failure, got %v", err)
	}
	if len(builder.requests) != 0 {
		t.Fatal("source resolution failure must occur before executable build")
	}
}

func TestRunnableManifestCompilerPreservesNonMainSourceFailure(t *testing.T) {
	t.Setenv("GOTELEMETRY", "off")
	workingDirectory, _ := runnablePackageRepositoryPaths(t)
	admission, sources := admitRunnableManifestForTest(
		t,
		runnableManifestApplication(testNonMainImportPath),
	)
	builder, err := NewGoApplicationExecutableBuilder(NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	runnableCompiler, err := NewRunnablePackageCompiler(
		sources,
		builder,
		NewZIPPackager(),
	)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewRunnableManifestCompiler(runnableCompiler)
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
	coordinator, err := NewRunnableManifestCompiler(nil)
	if !errors.Is(err, ErrExecutableBuildFailed) || coordinator != nil {
		t.Fatalf("expected nil compiler failure, got %#v, %v", coordinator, err)
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
	accessorCopy, ok := admission.ApplicationEntrypoint()
	if !ok {
		t.Fatal("expected admitted entrypoint")
	}
	accessorCopy.Module = "mutated-accessor-copy"
	accessorCopy.Version = "v8"

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

	signer, verifier := trustedTestSignerAndVerifier(t)
	builder, err := NewGoApplicationExecutableBuilder(NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	runnableCompiler, err := NewRunnablePackageCompiler(
		sources,
		builder,
		NewZIPPackagerWithSigner(signer),
	)
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := NewRunnableManifestCompiler(runnableCompiler)
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
