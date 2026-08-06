package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWatcherConcurrentRead(t *testing.T) {

	dir := t.TempDir()

	cfgFile := filepath.Join(dir, "forge.yaml")

	yaml := `
project:
  name: Forge
  version: dev

runtime:
  environment: development
  log_level: info

server:
  host: localhost
  port: 8080

plugins:
  logger:
    enabled: true
`

	if err := os.WriteFile(cfgFile, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(cfgFile)
	if err != nil {
		t.Fatal(err)
	}

	defer w.Close()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {

		wg.Add(1)

		go func() {

			defer wg.Done()

			_ = w.Config()

		}()

	}

	wg.Wait()
}
