package container

import "testing"

type AutoWireConfig struct {
	Name string
}

type AutoWireLogger struct {
	Config *AutoWireConfig
}

func TestAutoWire(t *testing.T) {

	c := New()

	cfg := &AutoWireConfig{
		Name: "Forge",
	}

	if err := c.Register(cfg); err != nil {
		t.Fatal(err)
	}

	logger := &AutoWireLogger{}

	if err := c.AutoWire(logger); err != nil {
		t.Fatal(err)
	}

	if logger.Config == nil {
		t.Fatal("config not injected")
	}

	if logger.Config.Name != "Forge" {
		t.Fatal("unexpected config")
	}
}
