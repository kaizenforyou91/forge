package compiler

import (
	"context"
	"fmt"
)

// ToolchainExecutor executes compiler commands through a CommandRunner.
type ToolchainExecutor struct {
	runner CommandRunner
}

// NewToolchainExecutor creates a toolchain executor.
func NewToolchainExecutor(runner CommandRunner) (*ToolchainExecutor, error) {
	if runner == nil {
		return nil, ErrNilCommandRunner
	}

	return &ToolchainExecutor{
		runner: runner,
	}, nil
}

// Execute executes one module compilation request.
func (e *ToolchainExecutor) Execute(
	request ExecutionRequest,
) (ExecutionResult, error) {
	if e == nil || e.runner == nil {
		return ExecutionResult{}, ErrNilCommandRunner
	}

	module, version, ok := splitModuleIdentity(request.Module)
	if !ok {
		return ExecutionResult{}, fmt.Errorf(
			"%w: invalid module identity %q",
			ErrInvalidBuildPlan,
			request.Module,
		)
	}

	args := []string{"build"}

	if request.ImportPath != "" {
		args = append(args, request.ImportPath)
	}

	command := Command{
		Name: "go",
		Args: args,
	}

	result, err := e.runner.Run(context.Background(), command)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf(
			"%w: %v",
			ErrCommandFailed,
			err,
		)
	}

	if result.ExitCode != 0 {
		return ExecutionResult{}, fmt.Errorf(
			"%w: go build exited with code %d: %s",
			ErrCommandFailed,
			result.ExitCode,
			result.Stderr,
		)
	}

	return ExecutionResult{
		Module:  module,
		Version: version,
	}, nil
}
