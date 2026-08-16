package compiler

import (
	"fmt"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

// ArtifactPackager packages a complete artifact bundle.
type ArtifactPackager interface {
	Package(
		bundle ArtifactBundle,
		payloads map[string][]byte,
		outputPath string,
	) error
}

// PackageArtifacts converts compiled artifacts into an ArtifactBundle
// and sends the package to the supplied ArtifactPackager.
func PackageArtifacts(
	plan manifest.BuildPlan,
	artifacts []Artifact,
	packager ArtifactPackager,
	outputPath string,
) error {
	if packager == nil {
		return fmt.Errorf(
			"%w: packager is nil",
			ErrInvalidArtifactPackage,
		)
	}

	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf(
			"%w: output path is required",
			ErrInvalidArtifactPackage,
		)
	}

	bundle, err := NewArtifactBundle(plan, artifacts)
	if err != nil {
		return err
	}

	payloads := make(map[string][]byte, len(artifacts))

	for _, artifact := range artifacts {
		key := artifact.Module + "@" + artifact.Version

		payloads[key] = append(
			[]byte(nil),
			artifactPayload(artifact)...,
		)
	}

	return packager.Package(
		bundle,
		payloads,
		outputPath,
	)
}

// CompileAndPackageManifest resolves a manifest, compiles it,
// and packages the resulting artifacts into one deterministic package.
func CompileAndPackageManifest(
	engine *Engine,
	m manifest.Manifest,
	packages *registry.Registry,
	packager ArtifactPackager,
	outputPath string,
) error {
	if engine == nil {
		return ErrNilExecutor
	}

	if packager == nil {
		return fmt.Errorf(
			"%w: packager is nil",
			ErrInvalidArtifactPackage,
		)
	}

	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf(
			"%w: output path is required",
			ErrInvalidArtifactPackage,
		)
	}

	resolved, err := manifest.ResolveDependencies(m, packages)
	if err != nil {
		return err
	}

	plan, err := manifest.BuildPlanForManifest(resolved)
	if err != nil {
		return err
	}

	artifacts, err := engine.Compile(plan)
	if err != nil {
		return err
	}

	return PackageArtifacts(
		plan,
		artifacts,
		packager,
		outputPath,
	)
}
