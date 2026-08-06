package config

import (
	"os"
	"testing"
)

func TestSaveHistory(t *testing.T) {

	cfg := Default()

	if err := SaveHistory(cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(".forge/config/history"); err != nil {
		t.Fatal(err)
	}
}
func TestListHistory(t *testing.T) {

	cfg := Default()

	if err := SaveHistory(cfg); err != nil {
		t.Fatal(err)
	}

	list, err := ListHistory()
	if err != nil {
		t.Fatal(err)
	}

	if len(list) == 0 {
		t.Fatal("history empty")
	}
}
