package config

import "testing"

func TestBeforeSaveHook(t *testing.T) {

	hooks = Hooks{}

	called := false

	RegisterBeforeSave(func(cfg *Config) error {

		called = true

		return nil
	})

	cfg := Default()

	if err := runHooks(hooks.BeforeSave, &cfg); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("hook not executed")
	}
}
