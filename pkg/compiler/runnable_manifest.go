package compiler

import (
	"context"
	"fmt"
)

type admittedApplicationSourceResolver struct {
	source PackageSource
}

func (r admittedApplicationSourceResolver) Resolve(
	name,
	version string,
) (PackageSource, error) {
	if name != r.source.Name || version != r.source.Version {
		return PackageSource{}, ErrPackageSourceNotFound
	}

	return r.source, nil
}

// RunnableManifestCompiler composes admitted manifest evidence into an
// admission-bound runnable package compiler request.
type RunnableManifestCompiler struct {
	builder  applicationExecutableBuilder
	packager runnablePackagePackager
}

// RunnableManifestRequest contains the caller-owned filesystem values needed
// to compile an already admitted runnable manifest.
type RunnableManifestRequest struct {
	Admission        ManifestAdmissionPlan
	WorkingDirectory string
	OutputPath       string
}

// NewRunnableManifestCompiler creates an admitted-manifest coordinator whose
// source authority is supplied exclusively by each admission request.
func NewRunnableManifestCompiler(
	builder *GoApplicationExecutableBuilder,
	packager *ZIPPackager,
) (*RunnableManifestCompiler, error) {
	if builder == nil {
		return nil, fmt.Errorf(
			"%w: executable builder is nil",
			ErrExecutableBuildFailed,
		)
	}
	if packager == nil {
		return nil, fmt.Errorf(
			"%w: runnable packager is nil",
			ErrInvalidArtifactPackage,
		)
	}

	return newRunnableManifestCompiler(
		builder,
		&zipRunnablePackagePackager{packager: packager},
	)
}

func newRunnableManifestCompiler(
	builder applicationExecutableBuilder,
	packager runnablePackagePackager,
) (*RunnableManifestCompiler, error) {
	if builder == nil {
		return nil, fmt.Errorf(
			"%w: executable builder is nil",
			ErrExecutableBuildFailed,
		)
	}
	if packager == nil {
		return nil, fmt.Errorf(
			"%w: runnable packager is nil",
			ErrInvalidArtifactPackage,
		)
	}

	return &RunnableManifestCompiler{
		builder:  builder,
		packager: packager,
	}, nil
}

// Compile converts the admitted application entrypoint exactly and delegates
// all build and package behavior to RunnablePackageCompiler.
func (c *RunnableManifestCompiler) Compile(
	ctx context.Context,
	request RunnableManifestRequest,
) error {
	if c == nil || c.builder == nil {
		return fmt.Errorf(
			"%w: runnable manifest compiler is incomplete",
			ErrExecutableBuildFailed,
		)
	}
	if c.packager == nil {
		return fmt.Errorf(
			"%w: runnable manifest compiler is incomplete",
			ErrInvalidArtifactPackage,
		)
	}

	entrypoint, present := request.Admission.ApplicationEntrypoint()
	if !present {
		return fmt.Errorf(
			"%w: admitted manifest has no application entrypoint",
			ErrInvalidApplicationEntrypoint,
		)
	}

	selectedSource, err := admittedApplicationSource(
		request.Admission,
		entrypoint.Module,
		entrypoint.Version,
	)
	if err != nil {
		return err
	}

	compiler, err := newRunnablePackageCompiler(
		admittedApplicationSourceResolver{source: selectedSource},
		c.builder,
		c.packager,
	)
	if err != nil {
		return err
	}

	return compiler.Compile(ctx, RunnablePackageRequest{
		Plan: request.Admission.BuildPlan(),
		Entrypoint: RuntimeEntrypoint{
			Module:  entrypoint.Module,
			Version: entrypoint.Version,
		},
		WorkingDirectory: request.WorkingDirectory,
		OutputPath:       request.OutputPath,
	})
}

func admittedApplicationSource(
	admission ManifestAdmissionPlan,
	module,
	version string,
) (PackageSource, error) {
	var selected PackageSource
	matches := 0

	for _, source := range admission.Sources() {
		if source.Name != module || source.Version != version {
			continue
		}

		selected = source
		matches++
	}

	if matches == 0 {
		return PackageSource{}, fmt.Errorf(
			"%w: admitted source for %s@%s: %w",
			ErrInvalidApplicationEntrypoint,
			module,
			version,
			ErrPackageSourceNotFound,
		)
	}
	if matches != 1 {
		return PackageSource{}, fmt.Errorf(
			"%w: admitted source for %s@%s matched %d candidates: %w",
			ErrInvalidApplicationEntrypoint,
			module,
			version,
			matches,
			ErrInvalidPackageSource,
		)
	}

	normalized, err := normalizePackageSource(selected)
	if err != nil || normalized != selected {
		return PackageSource{}, fmt.Errorf(
			"%w: admitted source for %s@%s is not canonical: %w",
			ErrInvalidApplicationEntrypoint,
			module,
			version,
			ErrInvalidPackageSource,
		)
	}

	return selected, nil
}
