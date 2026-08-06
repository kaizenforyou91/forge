package config

import "testing"

func TestWatcherLoad(t *testing.T) {
	cfg := Default()

	if cfg.Project.Name == "" {
		t.Fatal("config not loaded")
	}
}
