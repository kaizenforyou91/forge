package container

import "testing"

func TestGraph(t *testing.T) {

	c := New()

	err := c.RegisterConstructor(NewResolverArgsLogger)

	if err != nil {
		t.Fatal(err)
	}

	if c.graph == nil {
		t.Fatal("graph nil")
	}

	if len(c.graph.nodes) != 1 {
		t.Fatal("graph not populated")
	}

}
