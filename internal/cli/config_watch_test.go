package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigWatchCancelsCleanly(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "forge.yaml")

	yaml := `project:
  name: Forge
  version: dev

runtime:
  environment: development
  log_level: info

server:
  host: 127.0.0.1
  port: 8080

plugins:
  logger:
    enabled: true
`

	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newConfigCmd()
	pr, pw := io.Pipe()
	cmd.SetOut(pw)
	cmd.SetErr(pw)
	defer pw.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"watch", "--config", cfgPath})

	started := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		close(started)
		done <- cmd.Execute()
	}()

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("watch command did not start")
	}

	if err := waitForOutput(pr, fmt.Sprintf("Watching %s...", cfgPath), 3*time.Second); err != nil {
		t.Fatal(err)
	}

	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("expected canceled error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watch command did not exit after cancellation")
	}
}

func waitForOutput(r io.Reader, expected string, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		reader := bufio.NewReader(r)
		var acc strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				acc.WriteString(line)
				if strings.Contains(acc.String(), expected) {
					// keep draining in background to avoid blocking writer
					go io.Copy(io.Discard, reader)
					done <- nil
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					done <- fmt.Errorf("output closed before seeing %q", expected)
					return
				}
				done <- err
				return
			}
		}
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for output %q", expected)
	}
}
