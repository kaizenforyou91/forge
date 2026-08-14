package compiler

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

type trackingArtifactWriter struct {
	artifacts []Artifact
	payloads  [][]byte
	err       error
}

func (w *trackingArtifactWriter) Write(
	artifact Artifact,
	data []byte,
) error {
	if w.err != nil {
		return w.err
	}

	w.artifacts = append(w.artifacts, artifact)
	w.payloads = append(
		w.payloads,
		append([]byte(nil), data...),
	)

	return nil
}

func TestNewEngineWithWriter(t *testing.T) {
	executor := &writerTestExecutor{}
	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	if engine == nil {
		t.Fatal("expected engine")
	}

	if engine.executor != executor {
		t.Fatal("executor was not retained")
	}

	if engine.writer != writer {
		t.Fatal("writer was not retained")
	}
}

func TestNewEngineRemainsExecutionOnly(t *testing.T) {
	executor := &writerTestExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	if engine.writer != nil {
		t.Fatal("expected NewEngine to remain execution-only")
	}
}

func TestEngineWritesArtifactAfterSuccessfulExecution(t *testing.T) {
	executor := &writerTestExecutor{
		results: map[string]ExecutionResult{
			"web@v1": {
				Module:  "web",
				Version: "v1",
			},
		},
	}

	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	plan := manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "demo",
		Steps: []manifest.BuildStep{
			{
				Module: "web@v1",
			},
		},
	}

	artifacts, err := engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if len(artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(artifacts))
	}

	if len(writer.artifacts) != 1 {
		t.Fatalf(
			"expected 1 writer call, got %d",
			len(writer.artifacts),
		)
	}

	if !reflect.DeepEqual(
		writer.artifacts[0],
		artifacts[0],
	) {
		t.Fatalf(
			"writer received unexpected artifact: %#v",
			writer.artifacts[0],
		)
	}

	if string(writer.payloads[0]) != "web@v1" {
		t.Fatalf(
			"unexpected artifact payload %q",
			string(writer.payloads[0]),
		)
	}
}

func TestEngineWritesArtifactsInBuildPlanOrder(t *testing.T) {
	executor := &writerTestExecutor{
		results: map[string]ExecutionResult{
			"logger@v1": {
				Module:  "logger",
				Version: "v1",
			},
			"http@v1": {
				Module:  "http",
				Version: "v1",
			},
			"web@v1": {
				Module:  "web",
				Version: "v1",
			},
		},
	}

	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	plan := manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "demo",
		Steps: []manifest.BuildStep{
			{Module: "logger@v1"},
			{Module: "http@v1", Dependencies: []string{"logger@v1"}},
			{Module: "web@v1", Dependencies: []string{"http@v1", "logger@v1"}},
		},
	}

	_, err = engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"logger@v1",
		"http@v1",
		"web@v1",
	}

	if len(writer.artifacts) != len(want) {
		t.Fatalf(
			"expected %d writes, got %d",
			len(want),
			len(writer.artifacts),
		)
	}

	for i, artifact := range writer.artifacts {
		got := artifact.Module + "@" + artifact.Version

		if got != want[i] {
			t.Fatalf(
				"write %d: expected %q, got %q",
				i,
				want[i],
				got,
			)
		}
	}
}

func TestEngineDoesNotWriteAfterExecutorFailure(t *testing.T) {
	expectedErr := errors.New("compile failed")

	executor := &writerTestExecutor{
		results: map[string]ExecutionResult{
			"logger@v1": {
				Module:  "logger",
				Version: "v1",
			},
		},
		err: expectedErr,
	}

	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	plan := manifest.BuildPlan{
		Steps: []manifest.BuildStep{
			{Module: "logger@v1"},
		},
	}

	_, err = engine.Compile(plan)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected executor error, got %v",
			err,
		)
	}

	if len(writer.artifacts) != 0 {
		t.Fatalf(
			"expected no artifact writes, got %d",
			len(writer.artifacts),
		)
	}
}

func TestEngineStopsWhenWriterFails(t *testing.T) {
	executor := &writerTestExecutor{
		results: map[string]ExecutionResult{
			"logger@v1": {
				Module:  "logger",
				Version: "v1",
			},
			"http@v1": {
				Module:  "http",
				Version: "v1",
			},
		},
	}

	expectedErr := errors.New("write failed")

	writer := &trackingArtifactWriter{
		err: expectedErr,
	}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	plan := manifest.BuildPlan{
		Steps: []manifest.BuildStep{
			{Module: "logger@v1"},
			{Module: "http@v1"},
		},
	}

	_, err = engine.Compile(plan)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected writer error, got %v",
			err,
		)
	}
}

func TestEngineCopiesDependenciesBeforeExecution(t *testing.T) {
	executor := &dependencyMutatingExecutor{}

	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	plan := manifest.BuildPlan{
		Steps: []manifest.BuildStep{
			{
				Module:       "web@v1",
				Dependencies: []string{"http@v1", "logger@v1"},
			},
		},
	}

	_, err = engine.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"http@v1",
		"logger@v1",
	}

	if !reflect.DeepEqual(executor.dependencies, want) {
		t.Fatalf(
			"unexpected dependencies: %#v",
			executor.dependencies,
		)
	}

	if !reflect.DeepEqual(
		plan.Steps[0].Dependencies,
		want,
	) {
		t.Fatal("build plan dependencies were mutated")
	}
}

type dependencyMutatingExecutor struct {
	dependencies []string
}

func (e *dependencyMutatingExecutor) Execute(
	request ExecutionRequest,
) (ExecutionResult, error) {
	e.dependencies = append([]string(nil), request.Dependencies...)

	if len(request.Dependencies) > 0 {
		request.Dependencies[0] = "changed@v9"
	}

	return ExecutionResult{
		Module:  "web",
		Version: "v1",
	}, nil
}

type writerTestExecutor struct {
	results map[string]ExecutionResult
	err     error
}

func (e *writerTestExecutor) Execute(
	request ExecutionRequest,
) (ExecutionResult, error) {
	if e.err != nil {
		return ExecutionResult{}, e.err
	}

	if result, ok := e.results[request.Module]; ok {
		return result, nil
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
