package compiler

import (
	"context"
	"fmt"
)

type runnableManifestPackageCompiler interface {
	Compile(context.Context, RunnablePackageRequest) error
}

// RunnableManifestCompiler composes admitted manifest evidence into an
// existing runnable package compiler request.
type RunnableManifestCompiler struct {
	compiler runnableManifestPackageCompiler
}

// RunnableManifestRequest contains the caller-owned filesystem values needed
// to compile an already admitted runnable manifest.
type RunnableManifestRequest struct {
	Admission        ManifestAdmissionPlan
	WorkingDirectory string
	OutputPath       string
}

// NewRunnableManifestCompiler creates an admitted-manifest coordinator backed
// by an existing runnable package compiler.
func NewRunnableManifestCompiler(
	compiler *RunnablePackageCompiler,
) (*RunnableManifestCompiler, error) {
	if compiler == nil {
		return nil, fmt.Errorf(
			"%w: runnable package compiler is nil",
			ErrExecutableBuildFailed,
		)
	}

	return &RunnableManifestCompiler{compiler: compiler}, nil
}

// Compile converts the admitted application entrypoint exactly and delegates
// all build and package behavior to RunnablePackageCompiler.
func (c *RunnableManifestCompiler) Compile(
	ctx context.Context,
	request RunnableManifestRequest,
) error {
	if c == nil || c.compiler == nil {
		return fmt.Errorf(
			"%w: runnable manifest compiler is incomplete",
			ErrExecutableBuildFailed,
		)
	}

	entrypoint, present := request.Admission.ApplicationEntrypoint()
	if !present {
		return fmt.Errorf(
			"%w: admitted manifest has no application entrypoint",
			ErrInvalidApplicationEntrypoint,
		)
	}

	return c.compiler.Compile(ctx, RunnablePackageRequest{
		Plan: request.Admission.BuildPlan(),
		Entrypoint: RuntimeEntrypoint{
			Module:  entrypoint.Module,
			Version: entrypoint.Version,
		},
		WorkingDirectory: request.WorkingDirectory,
		OutputPath:       request.OutputPath,
	})
}
