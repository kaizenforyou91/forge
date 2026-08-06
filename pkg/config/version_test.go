package config

import "testing"

func TestCurrentVersion(t *testing.T) {

	if CurrentVersion <= 0 {
		t.Fatalf("invalid current version: %d", CurrentVersion)
	}
}

func TestCheckVersionValid(t *testing.T) {

	cfg := Default()
	cfg.Version = CurrentVersion

	if err := CheckVersion(cfg); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCheckVersionFuture(t *testing.T) {

	cfg := Default()
	cfg.Version = CurrentVersion + 1

	if err := CheckVersion(cfg); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpgrade(t *testing.T) {

	cfg := Default()
	cfg.Version = CurrentVersion

	if err := Upgrade(&cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Version != CurrentVersion {
		t.Fatalf(
			"expected version %d got %d",
			CurrentVersion,
			cfg.Version,
		)
	}
}
