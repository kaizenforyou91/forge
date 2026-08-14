package compiler

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestOSCommandRunnerContract(t *testing.T) {
	var _ CommandRunner = (*OSCommandRunner)(nil)
}

func TestOSCommandRunnerRunsGoVersion(t *testing.T) {
	runner := NewOSCommandRunner()

	result, err := runner.Run(
		context.Background(),
		Command{
			Name: "go",
			Args: []string{"version"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.ExitCode != 0 {
		t.Fatalf(
			"expected exit code 0, got %d, stderr=%q",
			result.ExitCode,
			result.Stderr,
		)
	}

	if !strings.Contains(result.Stdout, "go version") {
		t.Fatalf(
			"expected Go version output, got %q",
			result.Stdout,
		)
	}
}

func TestOSCommandRunnerRejectsEmptyCommand(t *testing.T) {
	runner := NewOSCommandRunner()

	_, err := runner.Run(
		context.Background(),
		Command{},
	)

	if !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf(
			"expected ErrInvalidCommand, got %v",
			err,
		)
	}
}

func TestOSCommandRunnerReportsNonZeroExit(t *testing.T) {
	runner := NewOSCommandRunner()

	result, err := runner.Run(
		context.Background(),
		Command{
			Name: "go",
			Args: []string{"help", "__forge_nonexistent_topic__"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.ExitCode == 0 {
		t.Fatalf(
			"expected non-zero exit code, got %d",
			result.ExitCode,
		)
	}

	if result.Stderr == "" && result.Stdout == "" {
		t.Fatal("expected command failure output")
	}
}

func TestOSCommandRunnerStartFailure(t *testing.T) {
	runner := NewOSCommandRunner()

	_, err := runner.Run(
		context.Background(),
		Command{
			Name: "forge-command-that-does-not-exist",
		},
	)

	if err == nil {
		t.Fatal("expected command start error")
	}
}

func TestOSCommandRunnerSupportsCancellation(t *testing.T) {
	runner := NewOSCommandRunner()

	if runtime.GOOS == "windows" {
		t.Skip("portable cancellation command is not stable across Windows process semantics")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		50*time.Millisecond,
	)
	defer cancel()

	result, err := runner.Run(
		ctx,
		Command{
			Name: "sleep",
			Args: []string{"5"},
		},
	)

	if err != nil {
		t.Fatalf("unexpected runner error: %v", err)
	}

	if result.ExitCode == 0 {
		t.Fatal("expected cancelled command to return non-zero exit code")
	}

	if ctx.Err() == nil {
		t.Fatal("expected context cancellation")
	}
}
