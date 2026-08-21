package compiler

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

type fakeExecutor struct {
	requests []ExecutionRequest
	err      error
}

type provenanceExecutor struct{}

func (provenanceExecutor) Execute(ExecutionRequest) (ExecutionResult, error) {
	return ExecutionResult{
		Module:     "compiler",
		Version:    "v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	}, nil
}

func (f *fakeExecutor) Execute(
	request ExecutionRequest,
) (ExecutionResult, error) {
	f.requests = append(
		f.requests,
		ExecutionRequest{
			Module:       request.Module,
			Dependencies: append([]string(nil), request.Dependencies...),
		},
	)

	if f.err != nil {
		return ExecutionResult{}, f.err
	}

	module, version, ok := splitModuleIdentity(request.Module)
	if !ok {
		return ExecutionResult{}, ErrInvalidBuildPlan
	}

	return ExecutionResult{
		Module:  module,
		Version: version,
	}, nil
}

func validExecutionPlan() manifest.BuildPlan {
	return manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "demo",
		Steps: []manifest.BuildStep{
			{
				Module: "logger@v1",
			},
			{
				Module: "http@v1",
				Dependencies: []string{
					"logger@v1",
				},
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

func TestNewEngine(t *testing.T) {
	executor := &fakeExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	if engine == nil {
		t.Fatal("expected engine")
	}
}

func TestNewEngineRejectsNilExecutor(t *testing.T) {
	_, err := NewEngine(nil)

	if !errors.Is(err, ErrNilExecutor) {
		t.Fatalf("expected ErrNilExecutor, got %v", err)
	}
}

func TestEngineCompilesInBuildPlanOrder(t *testing.T) {
	executor := &fakeExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	plan := validExecutionPlan()

	artifacts, err := engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	wantRequests := []ExecutionRequest{
		{
			Module: "logger@v1",
		},
		{
			Module: "http@v1",
			Dependencies: []string{
				"logger@v1",
			},
		},
		{
			Module: "web@v1",
			Dependencies: []string{
				"http@v1",
				"logger@v1",
			},
		},
	}

	if !reflect.DeepEqual(executor.requests, wantRequests) {
		t.Fatalf(
			"unexpected execution requests:\nwant %#v\ngot  %#v",
			wantRequests,
			executor.requests,
		)
	}

	wantArtifacts := []Artifact{
		{
			Module:  "logger",
			Version: "v1",
		},
		{
			Module:  "http",
			Version: "v1",
		},
		{
			Module:  "web",
			Version: "v1",
		},
	}

	if !reflect.DeepEqual(artifacts, wantArtifacts) {
		t.Fatalf(
			"unexpected artifacts:\nwant %#v\ngot  %#v",
			wantArtifacts,
			artifacts,
		)
	}
}

func TestEnginePreservesArtifactImportPath(t *testing.T) {
	engine, err := NewEngine(provenanceExecutor{})
	if err != nil {
		t.Fatal(err)
	}

	plan := manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "demo",
		Steps: []manifest.BuildStep{
			{Module: "compiler@v1"},
		},
	}

	artifacts, err := engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}

	artifact := artifacts[0]

	if artifact.Module != "compiler" {
		t.Fatalf("expected module %q, got %q", "compiler", artifact.Module)
	}

	if artifact.Version != "v1" {
		t.Fatalf("expected version %q, got %q", "v1", artifact.Version)
	}

	if artifact.ImportPath != "github.com/kaizenforyou91/forge/pkg/compiler" {
		t.Fatalf(
			"expected import path %q, got %q",
			"github.com/kaizenforyou91/forge/pkg/compiler",
			artifact.ImportPath,
		)
	}
}

func TestEnginePreservesDependencyMetadata(t *testing.T) {
	executor := &fakeExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	plan := validExecutionPlan()

	_, err = engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"http@v1",
		"logger@v1",
	}

	if !reflect.DeepEqual(
		executor.requests[2].Dependencies,
		want,
	) {
		t.Fatalf(
			"unexpected dependency metadata:\nwant %#v\ngot  %#v",
			want,
			executor.requests[2].Dependencies,
		)
	}
}

func TestEnginePropagatesExecutorError(t *testing.T) {
	expectedErr := errors.New("execution failed")

	executor := &fakeExecutor{
		err: expectedErr,
	}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.Compile(validExecutionPlan())
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected executor error, got %v",
			err,
		)
	}
}

func TestEngineStopsAtFirstExecutorError(t *testing.T) {
	executor := &failingAfterFirstExecutor{
		failOnCall: 1,
		err:        errors.New("second execution failed"),
	}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.Compile(validExecutionPlan())
	if !errors.Is(err, executor.err) {
		t.Fatalf(
			"expected executor error, got %v",
			err,
		)
	}

	if executor.calls != 2 {
		t.Fatalf(
			"expected 2 executor calls, got %d",
			executor.calls,
		)
	}
}

type failingAfterFirstExecutor struct {
	calls      int
	failOnCall int
	err        error
}

func (f *failingAfterFirstExecutor) Execute(
	request ExecutionRequest,
) (ExecutionResult, error) {
	f.calls++

	if f.calls == f.failOnCall+1 {
		return ExecutionResult{}, f.err
	}

	module, version, ok := splitModuleIdentity(request.Module)
	if !ok {
		return ExecutionResult{}, ErrInvalidBuildPlan
	}

	return ExecutionResult{
		Module:  module,
		Version: version,
	}, nil
}

func TestEngineRejectsMalformedModuleIdentity(t *testing.T) {
	executor := &fakeExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	plan := manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "invalid",
		Steps: []manifest.BuildStep{
			{
				Module: "invalid-module",
			},
		},
	}

	_, err = engine.Compile(plan)
	if !errors.Is(err, ErrInvalidBuildPlan) {
		t.Fatalf(
			"expected ErrInvalidBuildPlan, got %v",
			err,
		)
	}

	if len(executor.requests) != 0 {
		t.Fatal("executor should not run for malformed module identity")
	}
}

func TestEngineAllowsEmptyBuildPlan(t *testing.T) {
	executor := &fakeExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	plan := manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "empty",
	}

	artifacts, err := engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if len(artifacts) != 0 {
		t.Fatalf(
			"expected 0 artifacts, got %d",
			len(artifacts),
		)
	}

	if len(executor.requests) != 0 {
		t.Fatalf(
			"expected 0 execution requests, got %d",
			len(executor.requests),
		)
	}
}

func TestEngineDoesNotMutateBuildPlan(t *testing.T) {
	executor := &fakeExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	plan := validExecutionPlan()
	original := plan

	_, err = engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(plan, original) {
		t.Fatal("engine mutated build plan")
	}
}

func TestEngineCopiesDependencyMetadata(t *testing.T) {
	executor := &mutatingExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	plan := validExecutionPlan()

	_, err = engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(
		plan.Steps[1].Dependencies,
		[]string{"logger@v1"},
	) {
		t.Fatal("executor mutated caller-owned dependency data")
	}
}

type mutatingExecutor struct{}

func (m *mutatingExecutor) Execute(
	request ExecutionRequest,
) (ExecutionResult, error) {
	if len(request.Dependencies) > 0 {
		request.Dependencies[0] = "mutated@v9"
	}

	module, version, ok := splitModuleIdentity(request.Module)
	if !ok {
		return ExecutionResult{}, ErrInvalidBuildPlan
	}

	return ExecutionResult{
		Module:  module,
		Version: version,
	}, nil
}

func TestEngineProducesDeterministicOutput(t *testing.T) {
	plan := validExecutionPlan()

	firstExecutor := &fakeExecutor{}
	firstEngine, err := NewEngine(firstExecutor)
	if err != nil {
		t.Fatal(err)
	}

	first, err := firstEngine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	secondExecutor := &fakeExecutor{}
	secondEngine, err := NewEngine(secondExecutor)
	if err != nil {
		t.Fatal(err)
	}

	second, err := secondEngine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf(
			"compiler output is not deterministic:\nfirst %#v\nsecond %#v",
			first,
			second,
		)
	}
}
