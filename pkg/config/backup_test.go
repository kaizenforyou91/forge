package config

import (
	"os"
	"testing"
)

func TestBackup(t *testing.T) {

	file := "backup.yaml"

	cfg := Default()

	if err := Save(file, cfg); err != nil {
		t.Fatal(err)
	}

	defer os.Remove(file)
	defer os.Remove(file + ".bak")

	if err := Backup(file); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file + ".bak"); err != nil {
		t.Fatal("backup file not created")
	}
}
func TestRestore(t *testing.T) {

	file := "restore.yaml"

	cfg := Default()

	if err := Save(file, cfg); err != nil {
		t.Fatal(err)
	}

	if err := Backup(file); err != nil {
		t.Fatal(err)
	}

	defer os.Remove(file)
	defer os.Remove(file + ".bak")

	if err := Restore(file); err != nil {
		t.Fatal(err)
	}
}
