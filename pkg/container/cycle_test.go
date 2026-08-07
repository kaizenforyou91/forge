package container

import "testing"

func TestDependencyGraphValidate(t *testing.T) {

	g := &DependencyGraph{
		Nodes: map[string][]string{
			"A": {"B"},
			"B": {"C"},
			"C": {},
		},
	}

	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}

}

func TestDependencyGraphCycle(t *testing.T) {

	g := &DependencyGraph{
		Nodes: map[string][]string{
			"A": {"B"},
			"B": {"C"},
			"C": {"A"},
		},
	}

	if err := g.Validate(); err == nil {
		t.Fatal("expected cycle")
	}

}
