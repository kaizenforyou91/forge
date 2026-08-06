package config

import "testing"

func TestRegisterMigration(t *testing.T) {

	migrations = nil

	RegisterMigration(Migration{
		From: 1,
		To:   2,
		Run: func(cfg *Config) error {
			cfg.Version = 2
			return nil
		},
	})

	cfg := Default()
	cfg.Version = 1

	if err := Migrate(&cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Version != 2 {
		t.Fatalf("expected version 2, got %d", cfg.Version)
	}
}
