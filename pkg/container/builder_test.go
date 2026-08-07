package container

import "testing"

type BuilderConfig struct {
	Name string
}

func TestBuilder(t *testing.T) {

	cfg := &BuilderConfig{
		Name: "Forge",
	}

	c := NewBuilder().
		Singleton(cfg).
		Build()

	result, err := Make[*BuilderConfig](c)
	if err != nil {
		t.Fatal(err)
	}

	if result.Name != "Forge" {
		t.Fatal("unexpected value")
	}
}
