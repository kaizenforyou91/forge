package manifest

import (
	"errors"
	"strings"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

func TestBuildDependencyGraph(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	graph, err := BuildDependencyGraph(m)
	if err != nil {
		t.Fatal(err)
	}

	if len(graph.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(graph.Nodes))
	}

	webDependencies := graph.Dependencies("web", "v1")

	want := []string{
		"http@v1",
		"logger@v1",
	}

	if len(webDependencies) != len(want) {
		t.Fatalf(
			"expected %d dependencies, got %d",
			len(want),
			len(webDependencies),
		)
	}

	for i := range want {
		if webDependencies[i] != want[i] {
			t.Fatalf(
				"dependency %d: expected %q, got %q",
				i,
				want[i],
				webDependencies[i],
			)
		}
	}
}

func TestBuildDependencyGraphCreatesLeafNodes(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	graph, err := BuildDependencyGraph(m)
	if err != nil {
		t.Fatal(err)
	}

	if dependencies := graph.Dependencies("logger", "v1"); len(dependencies) != 0 {
		t.Fatalf(
			"expected leaf module to have 0 dependencies, got %d",
			len(dependencies),
		)
	}
}

func TestBuildDependencyGraphRejectsMissingDependency(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
				},
			},
		},
	}

	_, err := BuildDependencyGraph(m)
	if err == nil {
		t.Fatal("expected missing dependency error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
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

func TestDependencyGraphDetectsDirectCycle(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "cycle",
		Modules: []Module{
			{
				Name:    "a",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "b", Version: "v1"},
				},
			},
			{
				Name:    "b",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "a", Version: "v1"},
				},
			},
		},
	}

	_, err := BuildDependencyGraph(m)
	if err == nil {
		t.Fatal("expected cycle error")
	}

	if !strings.Contains(err.Error(), "circular module dependency detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDependencyGraphDetectsIndirectCycle(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "cycle",
		Modules: []Module{
			{
				Name:    "a",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "b", Version: "v1"},
				},
			},
			{
				Name:    "b",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "c", Version: "v1"},
				},
			},
			{
				Name:    "c",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "a", Version: "v1"},
				},
			},
		},
	}

	_, err := BuildDependencyGraph(m)
	if err == nil {
		t.Fatal("expected cycle error")
	}

	if !strings.Contains(err.Error(), "circular module dependency detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDependencyGraphDependenciesReturnsSnapshot(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string][]string{
			"web@v1": {
				"http@v1",
			},
		},
	}

	got := graph.Dependencies("web", "v1")
	got[0] = "changed"

	if graph.Nodes["web@v1"][0] != "http@v1" {
		t.Fatal("dependency snapshot mutated the graph")
	}
}

func TestDependencyGraphValidate(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string][]string{
			"web@v1": {
				"http@v1",
			},
			"http@v1": {
				"logger@v1",
			},
			"logger@v1": {},
		},
	}

	if err := graph.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDependencyGraphValidateRejectsCycle(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string][]string{
			"a@v1": {
				"b@v1",
			},
			"b@v1": {
				"c@v1",
			},
			"c@v1": {
				"a@v1",
			},
		},
	}

	if err := graph.Validate(); err == nil {
		t.Fatal("expected cycle validation error")
	}
}

func TestDependencyGraphCycleDiagnosticIsDeterministic(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string][]string{
			"z@v1": {"y@v1"},
			"y@v1": {"z@v1"},
			"b@v1": {"a@v1"},
			"a@v1": {"b@v1"},
		},
	}
	const want = "circular module dependency detected: b@v1 -> a@v1"

	for i := 0; i < 200; i++ {
		err := graph.Validate()
		if err == nil {
			t.Fatal("expected cycle validation error")
		}
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("iteration %d: expected %q, got %v", i, want, err)
		}
	}
}

func TestDependencyGraphValidateNilGraph(t *testing.T) {
	var graph *DependencyGraph

	err := graph.Validate()
	if err == nil {
		t.Fatal("expected nil graph error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T", err)
	}

	if forgeErr.Code != forgeerrors.CodeInternal {
		t.Fatalf(
			"expected error code %s, got %s",
			forgeerrors.CodeInternal,
			forgeErr.Code,
		)
	}
}

func TestDependencyGraphUnknownNodeReturnsEmpty(t *testing.T) {
	graph := &DependencyGraph{
		Nodes: map[string][]string{
			"web@v1": {},
		},
	}

	got := graph.Dependencies("missing", "v1")

	if got != nil && len(got) != 0 {
		t.Fatalf("expected empty dependencies, got %v", got)
	}
}
