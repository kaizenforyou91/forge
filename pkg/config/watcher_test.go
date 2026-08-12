package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "forge.yaml")

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

	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	defer w.Close()

	if w == nil {
		t.Fatal("watcher is nil")
	}

	if w.Config().Server.Port != 8080 {
		t.Fatalf("expected port 8080, got %d", w.Config().Server.Port)
	}
}

func TestWatcherCloseIdempotent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "forge.yaml")
	yaml := `project:
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

	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWatcherRunStopsAfterClose(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "forge.yaml")
	yaml := `project:
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

	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		w.Run()
		wg.Done()
	}()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	finished := make(chan struct{})
	go func() {
		wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("watcher run did not stop after close")
	}
}
