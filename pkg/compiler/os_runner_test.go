package compiler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const osCommandRunnerHelperEnvironment = "FORGE_OS_COMMAND_RUNNER_HELPER"

func TestOSCommandRunnerHelperProcess(t *testing.T) {
	if os.Getenv(osCommandRunnerHelperEnvironment) != "1" {
		return
	}

	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 || separator+1 >= len(os.Args) {
		os.Exit(2)
	}

	switch os.Args[separator+1] {
	case "working-directory":
		directory, err := os.Getwd()
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Fprint(os.Stdout, directory)
	case "environment":
		fmt.Fprintf(
			os.Stdout,
			"explicit=%s inherited=%s",
			os.Getenv("FORGE_OS_RUNNER_EXPLICIT"),
			os.Getenv("FORGE_OS_RUNNER_INHERITED_ONLY"),
		)
	case "stdio":
		fmt.Fprint(os.Stdout, "helper stdout")
		fmt.Fprint(os.Stderr, "helper stderr")
		os.Exit(7)
	default:
		os.Exit(2)
	}

	os.Exit(0)
}

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

func TestOSCommandRunnerUsesExplicitWorkingDirectory(t *testing.T) {
	runner := NewOSCommandRunner()
	directory := t.TempDir()
	t.Setenv(osCommandRunnerHelperEnvironment, "1")

	result, err := runner.Run(
		context.Background(),
		Command{
			Name: os.Args[0],
			Args: []string{
				"-test.run=^TestOSCommandRunnerHelperProcess$",
				"--",
				"working-directory",
			},
			Dir: directory,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", result)
	}

	want, err := filepath.EvalSymlinks(directory)
	if err != nil {
		t.Fatal(err)
	}
	got, err := filepath.EvalSymlinks(strings.TrimSpace(result.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(filepath.Clean(got), filepath.Clean(want)) {
		t.Fatalf("expected working directory %q, got %q", want, got)
	}
}

func TestOSCommandRunnerUsesExplicitEnvironment(t *testing.T) {
	runner := NewOSCommandRunner()
	t.Setenv("FORGE_OS_RUNNER_INHERITED_ONLY", "parent")

	environment := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if found && (strings.EqualFold(key, osCommandRunnerHelperEnvironment) ||
			strings.EqualFold(key, "FORGE_OS_RUNNER_EXPLICIT") ||
			strings.EqualFold(key, "FORGE_OS_RUNNER_INHERITED_ONLY")) {
			continue
		}
		environment = append(environment, entry)
	}
	environment = append(
		environment,
		osCommandRunnerHelperEnvironment+"=1",
		"FORGE_OS_RUNNER_EXPLICIT=child",
	)

	result, err := runner.Run(
		context.Background(),
		Command{
			Name: os.Args[0],
			Args: []string{
				"-test.run=^TestOSCommandRunnerHelperProcess$",
				"--",
				"environment",
			},
			Env: environment,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", result)
	}
	if strings.TrimSpace(result.Stdout) != "explicit=child inherited=" {
		t.Fatalf("unexpected environment output %q", result.Stdout)
	}
}

func TestOSCommandRunnerNilEnvironmentInheritsEnvironment(t *testing.T) {
	runner := NewOSCommandRunner()
	t.Setenv(osCommandRunnerHelperEnvironment, "1")
	t.Setenv("FORGE_OS_RUNNER_EXPLICIT", "inherited")
	t.Setenv("FORGE_OS_RUNNER_INHERITED_ONLY", "parent")

	result, err := runner.Run(
		context.Background(),
		Command{
			Name: os.Args[0],
			Args: []string{
				"-test.run=^TestOSCommandRunnerHelperProcess$",
				"--",
				"environment",
			},
			Env: nil,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", result)
	}
	if strings.TrimSpace(result.Stdout) != "explicit=inherited inherited=parent" {
		t.Fatalf("unexpected inherited environment output %q", result.Stdout)
	}
}

func TestOSCommandRunnerEmptyEnvironmentDoesNotInheritEnvironment(t *testing.T) {
	runner := NewOSCommandRunner()
	t.Setenv(osCommandRunnerHelperEnvironment, "1")

	result, err := runner.Run(
		context.Background(),
		Command{
			Name: os.Args[0],
			Args: []string{
				"-test.run=^TestOSCommandRunnerHelperProcess$",
				"--",
				"environment",
			},
			Env: []string{},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected success, got %#v", result)
	}
	if strings.Contains(result.Stdout, "explicit=") || strings.Contains(result.Stdout, "inherited=") {
		t.Fatalf("expected empty environment not to inherit helper flag, got %q", result.Stdout)
	}
}

func TestOSCommandRunnerCapturesStdoutAndStderr(t *testing.T) {
	runner := NewOSCommandRunner()
	t.Setenv(osCommandRunnerHelperEnvironment, "1")

	result, err := runner.Run(
		context.Background(),
		Command{
			Name: os.Args[0],
			Args: []string{
				"-test.run=^TestOSCommandRunnerHelperProcess$",
				"--",
				"stdio",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 7 {
		t.Fatalf("expected exit code 7, got %d", result.ExitCode)
	}
	if result.Stdout != "helper stdout" {
		t.Fatalf("unexpected stdout %q", result.Stdout)
	}
	if result.Stderr != "helper stderr" {
		t.Fatalf("unexpected stderr %q", result.Stderr)
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
