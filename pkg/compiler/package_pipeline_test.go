package compiler

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

type fakeArtifactPackager struct {
	called     bool
	calls      int
	bundle     ArtifactBundle
	payloads   map[string][]byte
	outputPath string
	err        error
}

func (p *fakeArtifactPackager) Package(
	bundle ArtifactBundle,
	payloads map[string][]byte,
	outputPath string,
) error {
	p.called = true
	p.calls++
	p.bundle = bundle
	p.payloads = payloads
	p.outputPath = outputPath
	return p.err
}

type packagePipelineTestExecutor struct {
	requests           []ExecutionRequest
	err                error
	mutateDependencies bool
}

func (e *packagePipelineTestExecutor) Execute(
	request ExecutionRequest,
) (ExecutionResult, error) {
	e.requests = append(e.requests, ExecutionRequest{
		Module:       request.Module,
		Dependencies: append([]string(nil), request.Dependencies...),
		ImportPath:   request.ImportPath,
	})

	if e.mutateDependencies && len(request.Dependencies) > 0 {
		request.Dependencies[0] = "mutated@v9"
	}

	if e.err != nil {
		return ExecutionResult{}, e.err
	}

	module, version, ok := splitModuleIdentity(request.Module)
	if !ok {
		return ExecutionResult{}, ErrInvalidBuildPlan
	}

	return ExecutionResult{
		Module:     module,
		Version:    version,
		ImportPath: "example.com/forge/" + module,
	}, nil
}

func packagePipelineTestPlan() manifest.BuildPlan {
	return manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{
				Module: "logger@v1",
			},
			{
				Module:       "http@v1",
				Dependencies: []string{"logger@v1"},
			},
			{
				Module: "web@v1",
				Dependencies: []string{
					"http@v1",
					"logger@v1",
				},
			},
		},
	}
}

func newPackagePipelineTestEngine(
	t *testing.T,
	executor Executor,
) *Engine {
	t.Helper()

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	return engine
}

func clonePackagePipelinePlan(
	plan manifest.BuildPlan,
) manifest.BuildPlan {
	clone := plan
	clone.Steps = make([]manifest.BuildStep, len(plan.Steps))

	for i, step := range plan.Steps {
		clone.Steps[i] = step
		clone.Steps[i].Dependencies = append(
			[]string(nil),
			step.Dependencies...,
		)
	}

	return clone
}

func TestCompileAndPackagePlan(t *testing.T) {
	plan := packagePipelineTestPlan()
	executor := &packagePipelineTestExecutor{}
	engine := newPackagePipelineTestEngine(t, executor)
	packager := &fakeArtifactPackager{}
	output := filepath.Join(t.TempDir(), "demo.zip")

	if err := CompileAndPackagePlan(
		engine,
		plan,
		packager,
		output,
	); err != nil {
		t.Fatal(err)
	}

	if packager.calls != 1 {
		t.Fatalf("expected 1 packager call, got %d", packager.calls)
	}

	if packager.outputPath != output {
		t.Fatalf(
			"expected output path %q, got %q",
			output,
			packager.outputPath,
		)
	}

	if packager.bundle.ManifestName != plan.ManifestName ||
		packager.bundle.ManifestVersion != plan.ManifestVersion {
		t.Fatalf(
			"expected bundle identity %s@%s, got %s@%s",
			plan.ManifestName,
			plan.ManifestVersion,
			packager.bundle.ManifestName,
			packager.bundle.ManifestVersion,
		)
	}

	wantArtifacts := []Artifact{
		{
			Module:     "logger",
			Version:    "v1",
			ImportPath: "example.com/forge/logger",
		},
		{
			Module:     "http",
			Version:    "v1",
			ImportPath: "example.com/forge/http",
		},
		{
			Module:     "web",
			Version:    "v1",
			ImportPath: "example.com/forge/web",
		},
	}

	if !reflect.DeepEqual(packager.bundle.Artifacts, wantArtifacts) {
		t.Fatalf(
			"expected artifacts %#v, got %#v",
			wantArtifacts,
			packager.bundle.Artifacts,
		)
	}

	for _, artifact := range wantArtifacts {
		key := artifact.Module + "@" + artifact.Version
		if string(packager.payloads[key]) != key {
			t.Fatalf("expected payload %q, got %q", key, packager.payloads[key])
		}
	}
}

func TestCompileAndPackagePlanPreservesDependencyFirstOrder(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "ordered",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "zeta@v1"},
			{
				Module:       "alpha@v1",
				Dependencies: []string{"zeta@v1"},
			},
		},
	}

	executor := &packagePipelineTestExecutor{}
	engine := newPackagePipelineTestEngine(t, executor)
	packager := &fakeArtifactPackager{}

	if err := CompileAndPackagePlan(
		engine,
		plan,
		packager,
		filepath.Join(t.TempDir(), "ordered.zip"),
	); err != nil {
		t.Fatal(err)
	}

	wantExecutionOrder := []string{"zeta@v1", "alpha@v1"}
	if len(executor.requests) != len(wantExecutionOrder) {
		t.Fatalf(
			"expected %d requests, got %d",
			len(wantExecutionOrder),
			len(executor.requests),
		)
	}

	for i, want := range wantExecutionOrder {
		if executor.requests[i].Module != want {
			t.Fatalf(
				"request %d: expected %q, got %q",
				i,
				want,
				executor.requests[i].Module,
			)
		}
	}

	wantArtifactOrder := []string{"zeta", "alpha"}
	if len(packager.bundle.Artifacts) != len(wantArtifactOrder) {
		t.Fatalf(
			"expected %d artifacts, got %d",
			len(wantArtifactOrder),
			len(packager.bundle.Artifacts),
		)
	}

	for i, want := range wantArtifactOrder {
		if packager.bundle.Artifacts[i].Module != want {
			t.Fatalf(
				"artifact %d: expected %q, got %q",
				i,
				want,
				packager.bundle.Artifacts[i].Module,
			)
		}
	}
}

func TestCompileAndPackagePlanPreservesDependencyMetadata(t *testing.T) {
	plan := packagePipelineTestPlan()
	original := clonePackagePipelinePlan(plan)
	executor := &packagePipelineTestExecutor{
		mutateDependencies: true,
	}
	engine := newPackagePipelineTestEngine(t, executor)
	packager := &fakeArtifactPackager{}

	if err := CompileAndPackagePlan(
		engine,
		plan,
		packager,
		filepath.Join(t.TempDir(), "metadata.zip"),
	); err != nil {
		t.Fatal(err)
	}

	want := []string{"http@v1", "logger@v1"}
	if !reflect.DeepEqual(executor.requests[2].Dependencies, want) {
		t.Fatalf(
			"expected dependencies %#v, got %#v",
			want,
			executor.requests[2].Dependencies,
		)
	}

	if !reflect.DeepEqual(plan, original) {
		t.Fatalf("pipeline mutated dependency metadata: %#v", plan)
	}
}

func TestCompileAndPackagePlanRejectsNilEngine(t *testing.T) {
	packager := &fakeArtifactPackager{}

	err := CompileAndPackagePlan(
		nil,
		packagePipelineTestPlan(),
		packager,
		filepath.Join(t.TempDir(), "demo.zip"),
	)

	if !errors.Is(err, ErrNilExecutor) {
		t.Fatalf("expected ErrNilExecutor, got %v", err)
	}

	if packager.called {
		t.Fatal("packager must not run after engine validation failure")
	}
}

func TestCompileAndPackagePlanRejectsNilPackager(t *testing.T) {
	executor := &packagePipelineTestExecutor{}
	engine := newPackagePipelineTestEngine(t, executor)

	err := CompileAndPackagePlan(
		engine,
		packagePipelineTestPlan(),
		nil,
		filepath.Join(t.TempDir(), "demo.zip"),
	)

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}

	if len(executor.requests) != 0 {
		t.Fatal("executor must not run after packager validation failure")
	}
}

func TestCompileAndPackagePlanRejectsEmptyOutputPath(t *testing.T) {
	tests := []struct {
		name       string
		outputPath string
	}{
		{
			name:       "empty",
			outputPath: "",
		},
		{
			name:       "whitespace",
			outputPath: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &packagePipelineTestExecutor{}
			engine := newPackagePipelineTestEngine(t, executor)
			packager := &fakeArtifactPackager{}

			err := CompileAndPackagePlan(
				engine,
				packagePipelineTestPlan(),
				packager,
				tt.outputPath,
			)

			if !errors.Is(err, ErrInvalidArtifactPackage) {
				t.Fatalf(
					"expected ErrInvalidArtifactPackage, got %v",
					err,
				)
			}

			if len(executor.requests) != 0 {
				t.Fatal("executor must not run after output validation failure")
			}

			if packager.called {
				t.Fatal("packager must not run after output validation failure")
			}
		})
	}
}

func TestCompileAndPackagePlanPropagatesExecutorError(t *testing.T) {
	wantErr := errors.New("execution failed")
	executor := &packagePipelineTestExecutor{err: wantErr}
	engine := newPackagePipelineTestEngine(t, executor)
	packager := &fakeArtifactPackager{}

	err := CompileAndPackagePlan(
		engine,
		packagePipelineTestPlan(),
		packager,
		filepath.Join(t.TempDir(), "demo.zip"),
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected executor error, got %v", err)
	}

	if packager.called {
		t.Fatal("packager must not run after executor failure")
	}
}

func TestCompileAndPackagePlanPropagatesPackagerError(t *testing.T) {
	wantErr := errors.New("packaging failed")
	executor := &packagePipelineTestExecutor{}
	engine := newPackagePipelineTestEngine(t, executor)
	packager := &fakeArtifactPackager{err: wantErr}

	err := CompileAndPackagePlan(
		engine,
		packagePipelineTestPlan(),
		packager,
		filepath.Join(t.TempDir(), "demo.zip"),
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected packager error, got %v", err)
	}

	if packager.calls != 1 {
		t.Fatalf("expected 1 packager call, got %d", packager.calls)
	}
}

func TestCompileAndPackagePlanSupportsEmptyPlan(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "empty",
		ManifestVersion: "v1",
	}
	executor := &packagePipelineTestExecutor{}
	engine := newPackagePipelineTestEngine(t, executor)
	packager := &fakeArtifactPackager{}

	if err := CompileAndPackagePlan(
		engine,
		plan,
		packager,
		filepath.Join(t.TempDir(), "empty.zip"),
	); err != nil {
		t.Fatal(err)
	}

	if len(executor.requests) != 0 {
		t.Fatalf("expected 0 requests, got %d", len(executor.requests))
	}

	if packager.calls != 1 {
		t.Fatalf("expected 1 packager call, got %d", packager.calls)
	}

	if len(packager.bundle.Artifacts) != 0 {
		t.Fatalf(
			"expected 0 artifacts, got %d",
			len(packager.bundle.Artifacts),
		)
	}

	if len(packager.payloads) != 0 {
		t.Fatalf("expected 0 payloads, got %d", len(packager.payloads))
	}
}

func TestCompileAndPackagePlanDoesNotMutatePlan(t *testing.T) {
	plan := packagePipelineTestPlan()
	original := clonePackagePipelinePlan(plan)
	executor := &packagePipelineTestExecutor{
		mutateDependencies: true,
	}
	engine := newPackagePipelineTestEngine(t, executor)

	if err := CompileAndPackagePlan(
		engine,
		plan,
		&fakeArtifactPackager{},
		filepath.Join(t.TempDir(), "demo.zip"),
	); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(plan, original) {
		t.Fatalf(
			"expected unchanged plan %#v, got %#v",
			original,
			plan,
		)
	}
}

func TestCompileAndPackageManifestDelegatesWithoutBehaviorChange(
	t *testing.T,
) {
	m := manifest.Manifest{
		Name:    "demo",
		Version: "v1",
		Modules: []manifest.Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []manifest.Dependency{
					{Name: "http", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}

	packages := pipelineTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
	)
	executor := &packagePipelineTestExecutor{}
	engine := newPackagePipelineTestEngine(t, executor)
	packager := &fakeArtifactPackager{}
	output := filepath.Join(t.TempDir(), "demo.zip")

	if err := CompileAndPackageManifest(
		engine,
		m,
		packages,
		packager,
		output,
	); err != nil {
		t.Fatal(err)
	}

	wantOrder := []string{"http@v1", "web@v1"}
	if len(executor.requests) != len(wantOrder) {
		t.Fatalf(
			"expected %d requests, got %d",
			len(wantOrder),
			len(executor.requests),
		)
	}

	for i, want := range wantOrder {
		if executor.requests[i].Module != want {
			t.Fatalf(
				"request %d: expected %q, got %q",
				i,
				want,
				executor.requests[i].Module,
			)
		}

		artifact := packager.bundle.Artifacts[i]
		if artifact.Module+"@"+artifact.Version != want {
			t.Fatalf(
				"artifact %d: expected %q, got %q",
				i,
				want,
				artifact.Module+"@"+artifact.Version,
			)
		}
	}

	if err := packager.bundle.Validate(); err != nil {
		t.Fatalf("expected valid output bundle, got %v", err)
	}

	if packager.outputPath != output {
		t.Fatalf(
			"expected output path %q, got %q",
			output,
			packager.outputPath,
		)
	}
}

func TestPackageArtifactsBuildsBundleAndPayloads(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{
				Module: "http@v1",
			},
			{
				Module: "web@v1",
			},
		},
	}

	artifacts := []Artifact{
		{
			Module:     "http",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			Version:    "v1",
		},
		{
			Module:     "web",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
			Version:    "v1",
		},
	}

	packager := &fakeArtifactPackager{}
	output := filepath.Join(t.TempDir(), "demo.zip")

	if err := PackageArtifacts(
		plan,
		artifacts,
		packager,
		output,
	); err != nil {
		t.Fatal(err)
	}

	if !packager.called {
		t.Fatal("expected packager to be called")
	}

	if packager.outputPath != output {
		t.Fatalf(
			"expected output path %q, got %q",
			output,
			packager.outputPath,
		)
	}

	if packager.bundle.ManifestName != "demo" {
		t.Fatalf(
			"expected manifest name demo, got %q",
			packager.bundle.ManifestName,
		)
	}

	if len(packager.bundle.Artifacts) != 2 {
		t.Fatalf(
			"expected 2 artifacts, got %d",
			len(packager.bundle.Artifacts),
		)
	}

	if string(packager.payloads["http@v1"]) != "http@v1" {
		t.Fatal("unexpected http payload")
	}

	if string(packager.payloads["web@v1"]) != "web@v1" {
		t.Fatal("unexpected web payload")
	}
}

func TestPackageArtifactsRejectsNilPackager(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
	}

	err := PackageArtifacts(
		plan,
		nil,
		nil,
		filepath.Join(t.TempDir(), "demo.zip"),
	)

	if err == nil {
		t.Fatal("expected nil packager error")
	}
}

func TestPackageArtifactsRejectsArtifactMismatch(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{
				Module: "http@v1",
			},
		},
	}

	artifacts := []Artifact{
		{
			Module:     "web",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
			Version:    "v1",
		},
	}

	packager := &fakeArtifactPackager{}

	err := PackageArtifacts(
		plan,
		artifacts,
		packager,
		filepath.Join(t.TempDir(), "demo.zip"),
	)

	if err == nil {
		t.Fatal("expected artifact mismatch error")
	}

	if packager.called {
		t.Fatal("packager must not run after bundle validation failure")
	}
}
