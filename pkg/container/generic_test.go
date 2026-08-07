package container

import "testing"

type GenericConfig struct {
	Name string
}

func TestResolveAs(t *testing.T) {

	c := New()

	cfg := &GenericConfig{
		Name: "Forge",
	}

	if err := c.Register(cfg); err != nil {
		t.Fatal(err)
	}

	result, err := ResolveAs[*GenericConfig](c)
	if err != nil {
		t.Fatal(err)
	}

	if result == nil {
		t.Fatal("nil result")
	}

	if result.Name != "Forge" {
		t.Fatal("unexpected value")
	}
}
