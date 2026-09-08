package config

import (
	"os"
	"testing"
)

func TestSave(t *testing.T) {
	isolateConfigTest(t)

	file := "test_save.yaml"
	defer os.Remove(file)

	cfg := Default()

	if err := Save(file, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file); err != nil {
		t.Fatal("file not created")
	}
}
