package compiler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type executableBuildRequest struct {
	Entrypoint       RuntimeEntrypoint
	ImportPath       string
	WorkingDirectory string
	OutputPath       string
}

// ExecutableBuildResult describes a successfully built host application
// executable. The caller retains ownership of the output path and its cleanup.
type ExecutableBuildResult struct {
	Path       string
	TargetOS   string
	TargetArch string
	Entrypoint RuntimeEntrypoint
	ImportPath string
}

type applicationExecutableBuilder interface {
	Build(
		context.Context,
		executableBuildRequest,
	) (ExecutableBuildResult, error)
}

// GoApplicationExecutableBuilder builds one host Go application executable.
type GoApplicationExecutableBuilder struct {
	runner CommandRunner
}

// NewGoApplicationExecutableBuilder creates a Go application executable
// builder backed by the supplied command runner.
func NewGoApplicationExecutableBuilder(
	runner CommandRunner,
) (*GoApplicationExecutableBuilder, error) {
	if runner == nil {
		return nil, ErrNilCommandRunner
	}

	return &GoApplicationExecutableBuilder{runner: runner}, nil
}

// Build validates and builds one host application executable at the exact
// caller-supplied output path. The caller owns directory creation and cleanup.
func (b *GoApplicationExecutableBuilder) Build(
	ctx context.Context,
	request executableBuildRequest,
) (ExecutableBuildResult, error) {
	if b == nil || b.runner == nil {
		return ExecutableBuildResult{}, ErrNilCommandRunner
	}

	if err := validateExecutableBuildRequest(ctx, request); err != nil {
		return ExecutableBuildResult{}, err
	}

	if err := ctx.Err(); err != nil {
		return ExecutableBuildResult{}, fmt.Errorf(
			"%w: build context is not active: %w",
			ErrExecutableBuildFailed,
			err,
		)
	}

	environment := hostExecutableBuildEnvironment()

	listResult, err := b.runner.Run(ctx, Command{
		Name: "go",
		Args: []string{
			"list",
			"-f={{.Name}}",
			request.ImportPath,
		},
		Dir: request.WorkingDirectory,
		Env: environment,
	})
	if err != nil {
		return ExecutableBuildResult{}, executableCommandError(
			ctx,
			"go list",
			request.ImportPath,
			listResult,
			err,
		)
	}
	if listResult.ExitCode != 0 {
		return ExecutableBuildResult{}, executableCommandError(
			ctx,
			"go list",
			request.ImportPath,
			listResult,
			nil,
		)
	}

	if strings.TrimSpace(listResult.Stdout) != "main" {
		return ExecutableBuildResult{}, fmt.Errorf(
			"%w: Go package %q has package name %q, expected %q",
			ErrInvalidApplicationEntrypoint,
			request.ImportPath,
			strings.TrimSpace(listResult.Stdout),
			"main",
		)
	}

	buildResult, err := b.runner.Run(ctx, Command{
		Name: "go",
		Args: []string{
			"build",
			"-trimpath",
			"-buildvcs=false",
			"-o",
			request.OutputPath,
			request.ImportPath,
		},
		Dir: request.WorkingDirectory,
		Env: environment,
	})
	if err != nil {
		return ExecutableBuildResult{}, executableCommandError(
			ctx,
			"go build",
			request.ImportPath,
			buildResult,
			err,
		)
	}
	if buildResult.ExitCode != 0 {
		return ExecutableBuildResult{}, executableCommandError(
			ctx,
			"go build",
			request.ImportPath,
			buildResult,
			nil,
		)
	}

	if err := validateExecutableOutput(request.OutputPath); err != nil {
		return ExecutableBuildResult{}, err
	}

	return ExecutableBuildResult{
		Path:       request.OutputPath,
		TargetOS:   runtime.GOOS,
		TargetArch: runtime.GOARCH,
		Entrypoint: request.Entrypoint,
		ImportPath: request.ImportPath,
	}, nil
}

func validateExecutableBuildRequest(
	ctx context.Context,
	request executableBuildRequest,
) error {
	if ctx == nil {
		return fmt.Errorf("application executable build context is nil")
	}

	if strings.TrimSpace(request.Entrypoint.Module) == "" {
		return fmt.Errorf(
			"%w: entrypoint module is required",
			ErrInvalidApplicationEntrypoint,
		)
	}
	if strings.TrimSpace(request.Entrypoint.Version) == "" {
		return fmt.Errorf(
			"%w: entrypoint version is required",
			ErrInvalidApplicationEntrypoint,
		)
	}
	if strings.TrimSpace(request.ImportPath) == "" {
		return fmt.Errorf(
			"%w: entrypoint import path is required",
			ErrInvalidApplicationEntrypoint,
		)
	}

	if strings.TrimSpace(request.WorkingDirectory) == "" {
		return fmt.Errorf("application executable working directory is required")
	}
	workingDirectoryInfo, err := os.Stat(request.WorkingDirectory)
	if err != nil {
		return fmt.Errorf(
			"stat application executable working directory %q: %w",
			request.WorkingDirectory,
			err,
		)
	}
	if !workingDirectoryInfo.IsDir() {
		return fmt.Errorf(
			"application executable working directory %q is not a directory",
			request.WorkingDirectory,
		)
	}

	if strings.TrimSpace(request.OutputPath) == "" {
		return fmt.Errorf("application executable output path is required")
	}
	if !filepath.IsAbs(request.OutputPath) {
		return fmt.Errorf(
			"application executable output path %q must be absolute",
			request.OutputPath,
		)
	}

	outputParent := filepath.Dir(request.OutputPath)
	outputParentInfo, err := os.Stat(outputParent)
	if err != nil {
		return fmt.Errorf(
			"stat application executable output parent %q: %w",
			outputParent,
			err,
		)
	}
	if !outputParentInfo.IsDir() {
		return fmt.Errorf(
			"application executable output parent %q is not a directory",
			outputParent,
		)
	}

	if _, err := os.Lstat(request.OutputPath); err == nil {
		return fmt.Errorf(
			"application executable output path %q already exists",
			request.OutputPath,
		)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf(
			"inspect application executable output path %q: %w",
			request.OutputPath,
			err,
		)
	}

	return nil
}

func hostExecutableBuildEnvironment() []string {
	environment := os.Environ()
	result := make([]string, 0, len(environment)+2)

	for _, entry := range environment {
		key, _, found := strings.Cut(entry, "=")
		if found && (strings.EqualFold(key, "GOOS") || strings.EqualFold(key, "GOARCH")) {
			continue
		}

		result = append(result, entry)
	}

	result = append(
		result,
		"GOOS="+runtime.GOOS,
		"GOARCH="+runtime.GOARCH,
	)

	return result
}

func executableCommandError(
	ctx context.Context,
	operation,
	importPath string,
	result CommandResult,
	runnerErr error,
) error {
	if contextErr := ctx.Err(); contextErr != nil {
		if runnerErr != nil {
			return fmt.Errorf(
				"%w: %s %q: %v: %w",
				ErrExecutableBuildFailed,
				operation,
				importPath,
				runnerErr,
				contextErr,
			)
		}

		return fmt.Errorf(
			"%w: %s %q: %w",
			ErrExecutableBuildFailed,
			operation,
			importPath,
			contextErr,
		)
	}

	if runnerErr != nil {
		return fmt.Errorf(
			"%w: %s %q: %w",
			ErrExecutableBuildFailed,
			operation,
			importPath,
			runnerErr,
		)
	}

	return fmt.Errorf(
		"%w: %s %q exited with code %d: %s",
		ErrExecutableBuildFailed,
		operation,
		importPath,
		result.ExitCode,
		result.Stderr,
	)
}

func validateExecutableOutput(outputPath string) error {
	info, err := os.Lstat(outputPath)
	if err != nil {
		return fmt.Errorf(
			"%w: stat %q: %v",
			ErrExecutableOutputMissing,
			outputPath,
			err,
		)
	}

	return validateExecutableOutputInfo(outputPath, info)
}

func validateExecutableOutputInfo(outputPath string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf(
			"%w: %q is a symbolic link",
			ErrExecutableOutputMissing,
			outputPath,
		)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf(
			"%w: %q is not a regular file",
			ErrExecutableOutputMissing,
			outputPath,
		)
	}
	if info.Size() <= 0 {
		return fmt.Errorf(
			"%w: %q is empty",
			ErrExecutableOutputMissing,
			outputPath,
		)
	}

	return nil
}
