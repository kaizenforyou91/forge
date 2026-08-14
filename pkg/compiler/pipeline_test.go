package compiler

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

func pipelineTestRegistry(
	t *testing.T,
	packages ...registry.Package,
) *registry.Registry {
	t.Helper()

	r := registry.New()

	for _, pkg := range packages {
		if err := r.Register(pkg); err != nil {
			t.Fatal(err)
		}
	}

	return r
}

func TestCompileManifestEndToEnd(t *testing.T) {
	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
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
				Dependencies: []manifest.Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	r := pipelineTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
		registry.Package{Name: "logger", Version: "v1"},
	)

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

	artifacts, err := engine.CompileManifest(m, r)
	if err != nil {
		t.Fatal(err)
	}

	want := []Artifact{
		{Module: "logger", Version: "v1"},
		{Module: "http", Version: "v1"},
		{Module: "web", Version: "v1"},
	}

	if !reflect.DeepEqual(artifacts, want) {
		t.Fatalf(
			"expected artifacts %#v, got %#v",
			want,
			artifacts,
		)
	}

	if len(writer.artifacts) != len(want) {
		t.Fatalf(
			"expected %d written artifacts, got %d",
			len(want),
			len(writer.artifacts),
		)
	}

	for i := range want {
		if !reflect.DeepEqual(writer.artifacts[i], want[i]) {
			t.Fatalf(
				"writer artifact %d: expected %#v, got %#v",
				i,
				want[i],
				writer.artifacts[i],
			)
		}
	}
}

func TestCompileManifestUsesDependencyFirstOrder(t *testing.T) {
	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []manifest.Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []manifest.Dependency{
					{Name: "http", Version: "v1"},
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

	r := pipelineTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
		registry.Package{Name: "logger", Version: "v1"},
	)

	executor := &writerTestExecutor{
		results: map[string]ExecutionResult{
			"web@v1":    {Module: "web", Version: "v1"},
			"http@v1":   {Module: "http", Version: "v1"},
			"logger@v1": {Module: "logger", Version: "v1"},
		},
	}

	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.CompileManifest(m, r)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"http@v1",
		"logger@v1",
		"web@v1",
	}

	if len(writer.artifacts) != len(want) {
		t.Fatalf(
			"expected %d written artifacts, got %d",
			len(want),
			len(writer.artifacts),
		)
	}

	for i, artifact := range writer.artifacts {
		got := artifact.Module + "@" + artifact.Version

		if got != want[i] {
			t.Fatalf(
				"artifact %d: expected %q, got %q",
				i,
				want[i],
				got,
			)
		}
	}
}

func TestCompileManifestRejectsMissingRegistryPackageBeforeExecution(
	t *testing.T,
) {
	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
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

	r := pipelineTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
	)

	executor := &countingExecutor{}

	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.CompileManifest(m, r)
	if err == nil {
		t.Fatal("expected missing dependency error")
	}

	if executor.count != 0 {
		t.Fatalf(
			"expected executor not to run, got %d executions",
			executor.count,
		)
	}

	if len(writer.artifacts) != 0 {
		t.Fatalf(
			"expected writer not to run, got %d writes",
			len(writer.artifacts),
		)
	}
}

func TestCompileManifestRejectsCycleBeforeExecution(t *testing.T) {
	m := manifest.Manifest{
		Version: "v1",
		Name:    "cycle",
		Modules: []manifest.Module{
			{
				Name:    "a",
				Version: "v1",
				Dependencies: []manifest.Dependency{
					{Name: "b", Version: "v1"},
				},
			},
			{
				Name:    "b",
				Version: "v1",
				Dependencies: []manifest.Dependency{
					{Name: "a", Version: "v1"},
				},
			},
		},
	}

	r := pipelineTestRegistry(
		t,
		registry.Package{Name: "a", Version: "v1"},
		registry.Package{Name: "b", Version: "v1"},
	)

	executor := &countingExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.CompileManifest(m, r)
	if err == nil {
		t.Fatal("expected cycle error")
	}

	if executor.count != 0 {
		t.Fatalf(
			"expected executor not to run, got %d executions",
			executor.count,
		)
	}
}

func TestCompileManifestRejectsNilRegistryBeforeExecution(t *testing.T) {
	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []manifest.Module{
			{
				Name:    "web",
				Version: "v1",
			},
		},
	}

	executor := &countingExecutor{}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.CompileManifest(m, nil)
	if err == nil {
		t.Fatal("expected nil registry error")
	}

	if executor.count != 0 {
		t.Fatalf(
			"expected executor not to run, got %d executions",
			executor.count,
		)
	}
}

func TestCompileManifestPreservesDependencyMetadata(t *testing.T) {
	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []manifest.Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []manifest.Dependency{
					{Name: "http", Version: "v1"},
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

	r := pipelineTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
		registry.Package{Name: "logger", Version: "v1"},
	)

	executor := &dependencyCapturingExecutor{
		inner: writerTestExecutor{
			results: map[string]ExecutionResult{
				"web@v1":    {Module: "web", Version: "v1"},
				"http@v1":   {Module: "http", Version: "v1"},
				"logger@v1": {Module: "logger", Version: "v1"},
			},
		},
	}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.CompileManifest(m, r)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"web@v1": {
			"http@v1",
			"logger@v1",
		},
		"http@v1":   nil,
		"logger@v1": nil,
	}

	if !reflect.DeepEqual(executor.dependencies, want) {
		t.Fatalf(
			"dependency metadata mismatch: expected %#v, got %#v",
			want,
			executor.dependencies,
		)
	}
}

func TestCompileManifestDoesNotMutateManifest(t *testing.T) {
	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
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

	original := m

	r := pipelineTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
		registry.Package{Name: "http", Version: "v1"},
	)

	executor := &writerTestExecutor{
		results: map[string]ExecutionResult{
			"web@v1":  {Module: "web", Version: "v1"},
			"http@v1": {Module: "http", Version: "v1"},
		},
	}

	engine, err := NewEngine(executor)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.CompileManifest(m, r)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(m, original) {
		t.Fatal("CompileManifest mutated manifest")
	}
}

func TestCompileManifestPropagatesExecutorError(
	t *testing.T,
) {
	expectedErr := errors.New("compile failed")

	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []manifest.Module{
			{
				Name:    "web",
				Version: "v1",
			},
		},
	}

	r := pipelineTestRegistry(
		t,
		registry.Package{Name: "web", Version: "v1"},
	)

	executor := &writerTestExecutor{
		err: expectedErr,
	}

	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.CompileManifest(m, r)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected executor error, got %v",
			err,
		)
	}

	if len(writer.artifacts) != 0 {
		t.Fatalf(
			"expected writer not to run, got %d writes",
			len(writer.artifacts),
		)
	}
}

func TestCompileManifestSupportsEmptyModules(t *testing.T) {
	m := manifest.Manifest{
		Version: "v1",
		Name:    "empty",
	}

	r := registry.New()

	executor := &countingExecutor{}

	writer := &trackingArtifactWriter{}

	engine, err := NewEngineWithWriter(executor, writer)
	if err != nil {
		t.Fatal(err)
	}

	artifacts, err := engine.CompileManifest(m, r)
	if err != nil {
		t.Fatal(err)
	}

	if len(artifacts) != 0 {
		t.Fatalf(
			"expected 0 artifacts, got %d",
			len(artifacts),
		)
	}

	if executor.count != 0 {
		t.Fatalf(
			"expected 0 executions, got %d",
			executor.count,
		)
	}

	if len(writer.artifacts) != 0 {
		t.Fatalf(
			"expected 0 writes, got %d",
			len(writer.artifacts),
		)
	}
}

type countingExecutor struct {
	count int
}

func (e *countingExecutor) Execute(
	ExecutionRequest,
) (ExecutionResult, error) {
	e.count++

	return ExecutionResult{
		Module:  "unused",
		Version: "v1",
	}, nil
}

type dependencyCapturingExecutor struct {
	inner        writerTestExecutor
	dependencies map[string][]string
}

func (e *dependencyCapturingExecutor) Execute(
	request ExecutionRequest,
) (ExecutionResult, error) {
	if e.dependencies == nil {
		e.dependencies = make(map[string][]string)
	}

	e.dependencies[request.Module] =
		append([]string(nil), request.Dependencies...)

	return e.inner.Execute(request)
}
