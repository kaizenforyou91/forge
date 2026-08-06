package config

import "testing"

func TestCache(t *testing.T) {

	ClearCache()

	if _, ok := GetCache(); ok {
		t.Fatal("cache should be empty")
	}

	cfg := Default()

	SetCache(cfg)

	v, ok := GetCache()

	if !ok {
		t.Fatal("cache not found")
	}

	if v.Project.Name != cfg.Project.Name {
		t.Fatal("cache mismatch")
	}
}

func TestClearCache(t *testing.T) {

	SetCache(Default())

	ClearCache()

	if _, ok := GetCache(); ok {
		t.Fatal("cache should be cleared")
	}
}
