package compiler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
)

var ErrInvalidCommand = errors.New("invalid command")

// OSCommandRunner executes commands through the operating system.
type OSCommandRunner struct{}

// NewOSCommandRunner creates an operating-system command runner.
func NewOSCommandRunner() *OSCommandRunner {
	return &OSCommandRunner{}
}

// Run executes a command and captures stdout, stderr, and exit code.
func (r *OSCommandRunner) Run(
	ctx context.Context,
	command Command,
) (CommandResult, error) {
	if command.Name == "" {
		return CommandResult{}, ErrInvalidCommand
	}

	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(
		ctx,
		command.Name,
		command.Args...,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return CommandResult{}, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return CommandResult{}, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return CommandResult{}, err
	}

	stdoutData, stdoutErr := io.ReadAll(stdout)
	stderrData, stderrErr := io.ReadAll(stderr)

	waitErr := cmd.Wait()

	if stdoutErr != nil {
		return CommandResult{}, fmt.Errorf("read stdout: %w", stdoutErr)
	}

	if stderrErr != nil {
		return CommandResult{}, fmt.Errorf("read stderr: %w", stderrErr)
	}

	result := CommandResult{
		Stdout: string(stdoutData),
		Stderr: string(stderrData),
	}

	if waitErr == nil {
		result.ExitCode = 0
		return result, nil
	}

	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return result, waitErr
}
