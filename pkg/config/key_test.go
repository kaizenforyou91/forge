package config

import (
	"os"
	"testing"
)

func TestGenerateKey(t *testing.T) {

	os.Remove(keyFile)

	key, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}
}

func TestLoadKey(t *testing.T) {

	key, err := LoadKey()
	if err != nil {
		t.Fatal(err)
	}

	if len(key) != 32 {
		t.Fatal("invalid key")
	}
}
