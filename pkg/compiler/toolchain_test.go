package compiler

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestNewToolchainExecutorRejectsNilRunner(t *testing.T) {
	_, err := NewToolchainExecutor(nil)

	if !errors.Is(err, ErrNilCommandRunner) {
		t.Fatalf(
			"expected ErrNilCommandRunner, got %v",
			err,
		)
	}
}

func TestToolchainExecutorRunsGoBuild(t *testing.T) {
	runner := &fakeCommandRunner{
		result: CommandResult{
			ExitCode: 0,
			Stdout:   "ok",
		},
	}

	executor, err := NewToolchainExecutor(runner)
	if err != nil {
		t.Fatal(err)
	}

	result, err := executor.Execute(ExecutionRequest{
		Module: "web@v1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Module != "web" {
		t.Fatalf(
			"expected module %q, got %q",
			"web",
			result.Module,
		)
	}

	if result.Version != "v1" {
		t.Fatalf(
			"expected version %q, got %q",
			"v1",
			result.Version,
		)
	}

	want := []Command{
		{
			Name: "go",
			Args: []string{"build"},
		},
	}

	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf(
			"unexpected command:\nwant %#v\ngot  %#v",
			want,
			runner.commands,
		)
	}
}

func TestToolchainExecutorRejectsMalformedModule(t *testing.T) {
	runner := &fakeCommandRunner{}

	executor, err := NewToolchainExecutor(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = executor.Execute(ExecutionRequest{
		Module: "invalid",
	})

	if !errors.Is(err, ErrInvalidBuildPlan) {
		t.Fatalf(
			"expected ErrInvalidBuildPlan, got %v",
			err,
		)
	}

	if len(runner.commands) != 0 {
		t.Fatal("runner should not execute malformed module request")
	}
}

func TestToolchainExecutorPropagatesRunnerError(t *testing.T) {
	expectedErr := errors.New("runner failed")

	runner := &fakeCommandRunner{
		err: expectedErr,
	}

	executor, err := NewToolchainExecutor(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = executor.Execute(ExecutionRequest{
		Module: "web@v1",
	})

	if !errors.Is(err, ErrCommandFailed) {
		t.Fatalf(
			"expected ErrCommandFailed, got %v",
			err,
		)
	}
}

func TestToolchainExecutorRejectsNonZeroExitCode(t *testing.T) {
	runner := &fakeCommandRunner{
		result: CommandResult{
			ExitCode: 1,
			Stderr:   "build failed",
		},
	}

	executor, err := NewToolchainExecutor(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = executor.Execute(ExecutionRequest{
		Module: "web@v1",
	})

	if !errors.Is(err, ErrCommandFailed) {
		t.Fatalf(
			"expected ErrCommandFailed, got %v",
			err,
		)
	}
}

func TestToolchainExecutorDoesNotInvokeRunnerAfterFailure(t *testing.T) {
	runner := &fakeCommandRunner{}

	executor, err := NewToolchainExecutor(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = executor.Execute(ExecutionRequest{
		Module: "invalid",
	})

	if !errors.Is(err, ErrInvalidBuildPlan) {
		t.Fatalf(
			"expected ErrInvalidBuildPlan, got %v",
			err,
		)
	}

	if len(runner.commands) != 0 {
		t.Fatal("runner should not have been invoked")
	}

	_ = context.Background()
}

func TestToolchainExecutorUsesImportPath(t *testing.T) {
	runner := &fakeCommandRunner{
		result: CommandResult{
			ExitCode: 0,
		},
	}

	executor, err := NewToolchainExecutor(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = executor.Execute(ExecutionRequest{
		Module:     "compiler@v1",
		ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(runner.commands) != 1 {
		t.Fatalf(
			"expected 1 command, got %d",
			len(runner.commands),
		)
	}

	command := runner.commands[0]

	if command.Name != "go" {
		t.Fatalf("expected go command, got %q", command.Name)
	}

	want := []string{
		"build",
		"github.com/kaizenforyou91/forge/pkg/compiler",
	}

	if !reflect.DeepEqual(command.Args, want) {
		t.Fatalf(
			"expected args %#v, got %#v",
			want,
			command.Args,
		)
	}
}
