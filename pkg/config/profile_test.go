package config

import (
	"os"
	"testing"
)

func TestProfilePath(t *testing.T) {

	if ProfilePath("config", "") != "config.yaml" {
		t.Fatal("invalid default profile")
	}

	if ProfilePath("config", "development") != "config.development.yaml" {
		t.Fatal("invalid development profile")
	}

	if ProfilePath("config", "production") != "config.production.yaml" {
		t.Fatal("invalid production profile")
	}
}

func TestSaveLoadProfile(t *testing.T) {

	cfg := Default()

	if err := SaveProfile("config", "development", cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadProfile("config", "development")
	if err != nil {
		t.Fatal(err)
	}

	if loaded.Project.Name != cfg.Project.Name {
		t.Fatal("profile mismatch")
	}

	_ = os.Remove("config.development.yaml")
}
