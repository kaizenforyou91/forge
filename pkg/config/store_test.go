package config

import "testing"

func TestStoreSetGet(t *testing.T) {
	s := NewStore()

	s.Set("APP_NAME", "Forge")

	if got := s.Get("APP_NAME"); got != "Forge" {
		t.Fatalf("expected Forge, got %q", got)
	}
}

func TestStoreHas(t *testing.T) {
	s := NewStore()

	if s.Has("APP_NAME") {
		t.Fatal("key should not exist")
	}

	s.Set("APP_NAME", "Forge")

	if !s.Has("APP_NAME") {
		t.Fatal("key should exist")
	}
}

func TestStoreEnvironmentOverridesValue(t *testing.T) {
	s := NewStore()

	s.Set("APP_NAME", "Forge")

	t.Setenv("APP_NAME", "ForgeProduction")

	if got := s.Get("APP_NAME"); got != "ForgeProduction" {
		t.Fatalf("expected environment override, got %q", got)
	}
}

func TestStoreEnvironmentKeyExists(t *testing.T) {
	s := NewStore()

	t.Setenv("APP_ENV", "production")

	if !s.Has("APP_ENV") {
		t.Fatal("environment key should exist")
	}
}

func TestStoreMissingKey(t *testing.T) {
	s := NewStore()

	if got := s.Get("UNKNOWN_KEY"); got != "" {
		t.Fatalf("expected empty value, got %q", got)
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	s := NewStore()

	done := make(chan struct{})

	go func() {
		for i := 0; i < 1000; i++ {
			s.Set("APP_NAME", "Forge")
		}

		close(done)
	}()

	for i := 0; i < 1000; i++ {
		_ = s.Get("APP_NAME")
	}

	<-done
}
