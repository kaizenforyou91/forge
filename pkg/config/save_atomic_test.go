package config

import (
	"os"
	"testing"
)

func TestAtomicSave(t *testing.T) {

	file := "atomic.yaml"
	defer os.Remove(file)

	cfg := Default()

	if err := Save(file, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file + ".tmp"); err == nil {
		t.Fatal("temporary file still exists")
	}
}
