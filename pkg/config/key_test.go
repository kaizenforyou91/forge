package config

import (
	"bytes"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	isolateConfigTest(t)

	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestLoadKey(t *testing.T) {
	isolateConfigTest(t)

	want, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	key, err := LoadKey()
	if err != nil {
		t.Fatal(err)
	}

	if len(key) != 32 {
		t.Fatal("invalid key")
	}
	if !bytes.Equal(key, want) {
		t.Fatal("loaded key differs from the test key")
	}
}
