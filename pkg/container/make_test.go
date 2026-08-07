package container

import "testing"

type MakeConfig struct {
	Name string
}

func TestMake(t *testing.T) {

	c := New()

	cfg := &MakeConfig{
		Name: "Forge",
	}

	if err := c.Register(cfg); err != nil {
		t.Fatal(err)
	}

	result, err := Make[*MakeConfig](c)
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
