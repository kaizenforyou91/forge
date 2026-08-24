package compiler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

// RunnablePackageRequest describes one explicit runnable application package
// build. Package format, bundle schema, runtime kind, target, and executable
// filename are fixed internally by RunnablePackageCompiler.
type RunnablePackageRequest struct {
	Plan             manifest.BuildPlan
	Entrypoint       RuntimeEntrypoint
	WorkingDirectory string
	OutputPath       string
}

type runnablePackagePackager interface {
	packageRunnable(
		ArtifactBundle,
		map[string][]byte,
		string,
	) error
}

type zipRunnablePackagePackager struct {
	packager *ZIPPackager
}

func (p *zipRunnablePackagePackager) packageRunnable(
	bundle ArtifactBundle,
	payloads map[string][]byte,
	outputPath string,
) error {
	if p == nil || p.packager == nil {
		return fmt.Errorf("%w: runnable packager is nil", ErrInvalidArtifactPackage)
	}

	return p.packager.packageForMetadata(
		bundle,
		payloads,
		outputPath,
		packageMetadataV2(),
	)
}

// RunnablePackageCompiler builds one host application executable and packages
// its exact bytes as a single-artifact package-format-v2 archive.
type RunnablePackageCompiler struct {
	sources  PackageSourceResolver
	builder  applicationExecutableBuilder
	packager runnablePackagePackager
}

// NewRunnablePackageCompiler creates a runnable package compiler backed by the
// Go executable builder and a ZIP packager configured for optional signing.
func NewRunnablePackageCompiler(
	sources PackageSourceResolver,
	builder *GoApplicationExecutableBuilder,
	packager *ZIPPackager,
) (*RunnablePackageCompiler, error) {
	if builder == nil {
		return nil, fmt.Errorf("%w: executable builder is nil", ErrExecutableBuildFailed)
	}
	if packager == nil {
		return nil, fmt.Errorf("%w: runnable packager is nil", ErrInvalidArtifactPackage)
	}

	return newRunnablePackageCompiler(
		sources,
		builder,
		&zipRunnablePackagePackager{packager: packager},
	)
}

func newRunnablePackageCompiler(
	sources PackageSourceResolver,
	builder applicationExecutableBuilder,
	packager runnablePackagePackager,
) (*RunnablePackageCompiler, error) {
	if sources == nil {
		return nil, fmt.Errorf("%w: package source resolver is nil", ErrInvalidPackageSource)
	}
	if builder == nil {
		return nil, fmt.Errorf("%w: executable builder is nil", ErrExecutableBuildFailed)
	}
	if packager == nil {
		return nil, fmt.Errorf("%w: runnable packager is nil", ErrInvalidArtifactPackage)
	}

	return &RunnablePackageCompiler{
		sources:  sources,
		builder:  builder,
		packager: packager,
	}, nil
}

// Compile builds and packages one explicitly selected application entrypoint.
// Temporary executable state is owned and cleaned up by this coordinator.
func (c *RunnablePackageCompiler) Compile(
	ctx context.Context,
	request RunnablePackageRequest,
) (resultErr error) {
	if c == nil || c.sources == nil || c.builder == nil || c.packager == nil {
		return fmt.Errorf("%w: runnable package compiler is incomplete", ErrExecutableBuildFailed)
	}

	if err := validateRunnablePackageRequest(ctx, request); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf(
			"%w: runnable package context is not active: %w",
			ErrExecutableBuildFailed,
			err,
		)
	}

	if err := validateRunnableEntrypointMembership(
		request.Plan,
		request.Entrypoint,
	); err != nil {
		return err
	}

	source, err := c.sources.Resolve(
		request.Entrypoint.Module,
		request.Entrypoint.Version,
	)
	if err != nil {
		return fmt.Errorf(
			"%w: resolve entrypoint source %s@%s: %w",
			ErrInvalidApplicationEntrypoint,
			request.Entrypoint.Module,
			request.Entrypoint.Version,
			err,
		)
	}
	if source.Name != request.Entrypoint.Module ||
		source.Version != request.Entrypoint.Version ||
		strings.TrimSpace(source.ImportPath) == "" {
		return fmt.Errorf(
			"%w: source resolver returned invalid binding %#v for %s@%s",
			ErrInvalidApplicationEntrypoint,
			source,
			request.Entrypoint.Module,
			request.Entrypoint.Version,
		)
	}

	temporaryDirectory, err := os.MkdirTemp("", "forge-runnable-build-*")
	if err != nil {
		return fmt.Errorf(
			"%w: create runnable build directory: %w",
			ErrExecutableBuildFailed,
			err,
		)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(temporaryDirectory); cleanupErr != nil {
			wrapped := fmt.Errorf(
				"cleanup runnable build directory %q: %w",
				temporaryDirectory,
				cleanupErr,
			)
			if resultErr == nil {
				resultErr = wrapped
				return
			}

			resultErr = errors.Join(resultErr, wrapped)
		}
	}()

	executablePath := filepath.Join(
		temporaryDirectory,
		runnableExecutableFileName(),
	)
	buildRequest := executableBuildRequest{
		Entrypoint:       request.Entrypoint,
		ImportPath:       source.ImportPath,
		WorkingDirectory: request.WorkingDirectory,
		OutputPath:       executablePath,
	}

	result, err := c.builder.Build(ctx, buildRequest)
	if err != nil {
		if errors.Is(err, ErrInvalidApplicationEntrypoint) ||
			errors.Is(err, ErrExecutableBuildFailed) ||
			errors.Is(err, ErrExecutableOutputMissing) {
			return err
		}

		return fmt.Errorf(
			"%w: build entrypoint %s@%s: %w",
			ErrExecutableBuildFailed,
			request.Entrypoint.Module,
			request.Entrypoint.Version,
			err,
		)
	}

	if err := validateRunnableBuildResult(
		result,
		buildRequest,
	); err != nil {
		return err
	}
	if err := validateExecutableOutput(executablePath); err != nil {
		return err
	}

	executableBytes, err := os.ReadFile(executablePath)
	if err != nil {
		return fmt.Errorf(
			"%w: read executable output %q: %w",
			ErrExecutableOutputMissing,
			executablePath,
			err,
		)
	}
	if len(executableBytes) == 0 {
		return fmt.Errorf(
			"%w: executable output %q is empty after read",
			ErrExecutableOutputMissing,
			executablePath,
		)
	}

	artifact := Artifact{
		Module:     request.Entrypoint.Module,
		Version:    request.Entrypoint.Version,
		ImportPath: source.ImportPath,
	}
	bundle := ArtifactBundle{
		ManifestName:    request.Plan.ManifestName,
		ManifestVersion: request.Plan.ManifestVersion,
		Runtime: &RuntimeDescriptor{
			Kind:       RuntimeKindApplicationExecutable,
			Entrypoint: request.Entrypoint,
			TargetOS:   result.TargetOS,
			TargetArch: result.TargetArch,
		},
		Artifacts: []Artifact{artifact},
	}
	if err := bundle.ValidateForSchema(artifactBundleSchemaVersionV2); err != nil {
		return err
	}

	payloadKey := artifact.Module + "@" + artifact.Version
	payloads := map[string][]byte{
		payloadKey: append([]byte(nil), executableBytes...),
	}

	return c.packager.packageRunnable(
		bundle,
		payloads,
		request.OutputPath,
	)
}

func validateRunnablePackageRequest(
	ctx context.Context,
	request RunnablePackageRequest,
) error {
	if ctx == nil {
		return fmt.Errorf("runnable package context is nil")
	}
	if strings.TrimSpace(request.Plan.ManifestName) == "" ||
		strings.TrimSpace(request.Plan.ManifestVersion) == "" {
		return fmt.Errorf("%w: manifest identity is required", ErrInvalidBuildPlan)
	}

	for stepIndex, step := range request.Plan.Steps {
		if _, _, ok := splitModuleIdentity(step.Module); !ok {
			return fmt.Errorf(
				"%w: build step %d has invalid module identity %q",
				ErrInvalidBuildPlan,
				stepIndex,
				step.Module,
			)
		}

		for dependencyIndex, dependency := range step.Dependencies {
			if _, _, ok := splitModuleIdentity(dependency); !ok {
				return fmt.Errorf(
					"%w: build step %d dependency %d has invalid identity %q",
					ErrInvalidBuildPlan,
					stepIndex,
					dependencyIndex,
					dependency,
				)
			}
		}
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

	if strings.TrimSpace(request.WorkingDirectory) == "" {
		return fmt.Errorf("runnable package working directory is required")
	}
	workingDirectoryInfo, err := os.Stat(request.WorkingDirectory)
	if err != nil {
		return fmt.Errorf(
			"stat runnable package working directory %q: %w",
			request.WorkingDirectory,
			err,
		)
	}
	if !workingDirectoryInfo.IsDir() {
		return fmt.Errorf(
			"runnable package working directory %q is not a directory",
			request.WorkingDirectory,
		)
	}

	if strings.TrimSpace(request.OutputPath) == "" {
		return fmt.Errorf("%w: output path is required", ErrInvalidArtifactPackage)
	}

	return nil
}

func validateRunnableEntrypointMembership(
	plan manifest.BuildPlan,
	entrypoint RuntimeEntrypoint,
) error {
	matches := 0
	for _, step := range plan.Steps {
		module, version, _ := splitModuleIdentity(step.Module)
		if module == entrypoint.Module && version == entrypoint.Version {
			matches++
		}
	}

	if matches != 1 {
		return fmt.Errorf(
			"%w: entrypoint %s@%s must match exactly one build step, matched %d",
			ErrInvalidApplicationEntrypoint,
			entrypoint.Module,
			entrypoint.Version,
			matches,
		)
	}

	return nil
}

func validateRunnableBuildResult(
	result ExecutableBuildResult,
	request executableBuildRequest,
) error {
	if result.Path != request.OutputPath {
		return fmt.Errorf(
			"%w: executable builder returned path %q, expected %q",
			ErrExecutableBuildFailed,
			result.Path,
			request.OutputPath,
		)
	}
	if result.Entrypoint != request.Entrypoint {
		return fmt.Errorf(
			"%w: executable builder returned entrypoint %#v, expected %#v",
			ErrExecutableBuildFailed,
			result.Entrypoint,
			request.Entrypoint,
		)
	}
	if result.ImportPath != request.ImportPath {
		return fmt.Errorf(
			"%w: executable builder returned import path %q, expected %q",
			ErrExecutableBuildFailed,
			result.ImportPath,
			request.ImportPath,
		)
	}
	if strings.TrimSpace(result.TargetOS) == "" ||
		strings.TrimSpace(result.TargetArch) == "" {
		return fmt.Errorf(
			"%w: executable builder returned an empty target",
			ErrExecutableBuildFailed,
		)
	}
	if result.TargetOS != runtime.GOOS || result.TargetArch != runtime.GOARCH {
		return fmt.Errorf(
			"%w: executable builder returned target %s/%s, expected host %s/%s",
			ErrExecutableBuildFailed,
			result.TargetOS,
			result.TargetArch,
			runtime.GOOS,
			runtime.GOARCH,
		)
	}

	return nil
}

func runnableExecutableFileName() string {
	if runtime.GOOS == "windows" {
		return "application.exe"
	}

	return "application"
}
