package container

import "testing"

type ConstructorConfig struct {
	Name string
}

func TestResolveFromConstructor(t *testing.T) {

	c := New()

	err := c.RegisterFactory(func() any {
		return &ConstructorConfig{
			Name: "Forge",
		}
	})

	if err != nil {
		t.Fatal(err)
	}

	var cfg *ConstructorConfig

	err = c.Resolve(&cfg)

	if err != nil {
		t.Fatal(err)
	}

	if cfg == nil {
		t.Fatal("config is nil")
	}

	if cfg.Name != "Forge" {
		t.Fatal("unexpected value")
	}
}
