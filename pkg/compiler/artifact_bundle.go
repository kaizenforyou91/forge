package compiler

import (
	"fmt"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

// RuntimeKindApplicationExecutable identifies a statically compiled
// application executable runtime artifact.
const RuntimeKindApplicationExecutable = "application_executable"

// RuntimeDescriptor describes how a runnable artifact bundle is launched.
type RuntimeDescriptor struct {
	Kind       string
	Entrypoint RuntimeEntrypoint
	TargetOS   string
	TargetArch string
}

// RuntimeEntrypoint identifies the single artifact used as the application
// entrypoint. Archive paths remain derived from this logical identity.
type RuntimeEntrypoint struct {
	Module  string
	Version string
}

// ArtifactBundle groups the artifacts produced from one manifest build plan.
//
// Bundle identity is derived from the manifest identity.
// Artifact order is preserved exactly as produced by the build plan.
type ArtifactBundle struct {
	ManifestName    string
	ManifestVersion string
	Runtime         *RuntimeDescriptor
	Artifacts       []Artifact
}

// NewArtifactBundle creates a deterministic artifact bundle from a build plan
// and the artifacts produced for that plan.
//
// The number and order of artifacts must match the build plan steps.
func NewArtifactBundle(
	plan manifest.BuildPlan,
	artifacts []Artifact,
) (ArtifactBundle, error) {
	if plan.ManifestName == "" || plan.ManifestVersion == "" {
		return ArtifactBundle{}, fmt.Errorf(
			"%w: manifest identity is required",
			ErrInvalidArtifactBundle,
		)
	}

	if len(plan.Steps) != len(artifacts) {
		return ArtifactBundle{}, fmt.Errorf(
			"%w: build plan contains %d steps but received %d artifacts",
			ErrInvalidArtifactBundle,
			len(plan.Steps),
			len(artifacts),
		)
	}

	result := ArtifactBundle{
		ManifestName:    plan.ManifestName,
		ManifestVersion: plan.ManifestVersion,
		Artifacts:       make([]Artifact, len(artifacts)),
	}

	for i, artifact := range artifacts {
		if artifact.Module == "" ||
			artifact.Version == "" ||
			artifact.ImportPath == "" {
			return ArtifactBundle{}, fmt.Errorf(
				"%w: artifact %d has invalid identity",
				ErrInvalidArtifactBundle,
				i,
			)
		}

		if plan.Steps[i].Module != artifact.Module+"@"+artifact.Version {
			return ArtifactBundle{}, fmt.Errorf(
				"%w: artifact %d identity %q does not match build step %q",
				ErrInvalidArtifactBundle,
				i,
				artifact.Module+"@"+artifact.Version,
				plan.Steps[i].Module,
			)
		}

		result.Artifacts[i] = artifact
	}

	return result, nil
}

// Validate validates the artifact bundle contract.
func (b ArtifactBundle) Validate() error {
	return b.ValidateForSchema(artifactBundleSchemaVersionV1)
}

// ValidateForSchema validates an artifact bundle against a supported bundle
// schema contract.
func (b ArtifactBundle) ValidateForSchema(schemaVersion int) error {
	if schemaVersion != artifactBundleSchemaVersionV1 &&
		schemaVersion != artifactBundleSchemaVersionV2 {
		return fmt.Errorf(
			"%w: unsupported artifact bundle schema version %d",
			ErrInvalidArtifactBundle,
			schemaVersion,
		)
	}

	if b.ManifestName == "" {
		return fmt.Errorf(
			"%w: manifest name is required",
			ErrInvalidArtifactBundle,
		)
	}

	if b.ManifestVersion == "" {
		return fmt.Errorf(
			"%w: manifest version is required",
			ErrInvalidArtifactBundle,
		)
	}

	seen := make(map[string]struct{}, len(b.Artifacts))

	for i, artifact := range b.Artifacts {
		if artifact.Module == "" ||
			artifact.Version == "" ||
			artifact.ImportPath == "" {
			return fmt.Errorf(
				"%w: artifact %d has invalid identity",
				ErrInvalidArtifactBundle,
				i,
			)
		}

		key := artifact.Module + "@" + artifact.Version

		if _, exists := seen[key]; exists {
			return fmt.Errorf(
				"%w: duplicate artifact %q",
				ErrInvalidArtifactBundle,
				key,
			)
		}

		seen[key] = struct{}{}
	}

	if schemaVersion == artifactBundleSchemaVersionV1 {
		if b.Runtime != nil {
			return fmt.Errorf(
				"%w: runtime descriptor is not allowed in bundle schema version 1",
				ErrInvalidArtifactBundle,
			)
		}

		return nil
	}

	if b.Runtime == nil {
		return fmt.Errorf(
			"%w: runtime descriptor is required for bundle schema version 2",
			ErrInvalidArtifactBundle,
		)
	}

	runtime := b.Runtime
	if runtime.Kind != RuntimeKindApplicationExecutable {
		return fmt.Errorf(
			"%w: unsupported runtime kind %q",
			ErrInvalidArtifactBundle,
			runtime.Kind,
		)
	}
	if strings.TrimSpace(runtime.TargetOS) == "" {
		return fmt.Errorf("%w: runtime target OS is required", ErrInvalidArtifactBundle)
	}
	if strings.TrimSpace(runtime.TargetArch) == "" {
		return fmt.Errorf("%w: runtime target architecture is required", ErrInvalidArtifactBundle)
	}
	if strings.TrimSpace(runtime.Entrypoint.Module) == "" {
		return fmt.Errorf("%w: runtime entrypoint module is required", ErrInvalidArtifactBundle)
	}
	if strings.TrimSpace(runtime.Entrypoint.Version) == "" {
		return fmt.Errorf("%w: runtime entrypoint version is required", ErrInvalidArtifactBundle)
	}

	matches := 0
	for _, artifact := range b.Artifacts {
		if artifact.Module == runtime.Entrypoint.Module &&
			artifact.Version == runtime.Entrypoint.Version {
			matches++
		}
	}
	if matches != 1 {
		return fmt.Errorf(
			"%w: runtime entrypoint %q must match exactly one artifact, matched %d",
			ErrInvalidArtifactBundle,
			runtime.Entrypoint.Module+"@"+runtime.Entrypoint.Version,
			matches,
		)
	}

	return nil
}
