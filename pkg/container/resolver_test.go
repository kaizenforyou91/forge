package container

import "testing"

type ResolverArgsConfig struct{}

type ResolverArgsLogger struct {
	Config *ResolverArgsConfig
}

func NewResolverArgsLogger(cfg *ResolverArgsConfig) *ResolverArgsLogger {
	return &ResolverArgsLogger{Config: cfg}
}

func TestResolveArguments(t *testing.T) {

	c := New()

	cfg := &ResolverArgsConfig{}

	if err := c.Register(cfg); err != nil {
		t.Fatal(err)
	}

	constructor, err := ParseConstructor(NewResolverArgsLogger)
	if err != nil {
		t.Fatal(err)
	}

	args, err := c.resolveArguments(constructor)
	if err != nil {
		t.Fatal(err)
	}

	if len(args) != 1 {
		t.Fatal("expected one argument")
	}
}
