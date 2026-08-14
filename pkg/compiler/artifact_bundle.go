package compiler

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

// ArtifactBundle groups the artifacts produced from one manifest build plan.
//
// Bundle identity is derived from the manifest identity.
// Artifact order is preserved exactly as produced by the build plan.
type ArtifactBundle struct {
	ManifestName    string
	ManifestVersion string
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
		if artifact.Module == "" || artifact.Version == "" {
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
		if artifact.Module == "" || artifact.Version == "" {
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

	return nil
}
