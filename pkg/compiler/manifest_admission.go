package compiler

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

// ManifestAdmissionPlan contains the result of a successful manifest
// admission preflight.
//
// Its values are kept private so callers cannot mutate the prepared build plan
// or the package and source candidates intended for a later admission commit.
type ManifestAdmissionPlan struct {
	buildPlan manifest.BuildPlan
	packages  []registry.Package
	sources   []PackageSource
}

// BuildPlan returns an independent deep copy of the prepared build plan.
func (p ManifestAdmissionPlan) BuildPlan() manifest.BuildPlan {
	return cloneManifestAdmissionBuildPlan(p.buildPlan)
}

// Packages returns an independent copy of the manifest package candidates.
func (p ManifestAdmissionPlan) Packages() []registry.Package {
	return cloneManifestAdmissionPackages(p.packages)
}

// Sources returns an independent copy of the normalized source candidates.
func (p ManifestAdmissionPlan) Sources() []PackageSource {
	return cloneManifestAdmissionSources(p.sources)
}

// PrepareManifestAdmission validates and plans a manifest against temporary
// package and source overlays without mutating live registry state.
func PrepareManifestAdmission(
	m manifest.Manifest,
	existingPackages []registry.Package,
	existingSources []PackageSource,
) (ManifestAdmissionPlan, error) {
	if err := m.Validate(); err != nil {
		return ManifestAdmissionPlan{}, err
	}

	packages := make([]registry.Package, 0, len(m.Modules))
	sources := make([]PackageSource, 0, len(m.Modules))

	for i, module := range m.Modules {
		packages = append(packages, registry.Package{
			Name:    module.Name,
			Version: module.Version,
		})

		source, err := normalizePackageSource(PackageSource{
			Name:       module.Name,
			Version:    module.Version,
			ImportPath: module.ImportPath,
		})
		if err != nil {
			return ManifestAdmissionPlan{}, fmt.Errorf(
				"manifest module %d %q@%q source: %w",
				i,
				module.Name,
				module.Version,
				err,
			)
		}

		sources = append(sources, source)
	}

	packageOverlay := registry.New()
	if err := packageOverlay.EnsureAll(existingPackages); err != nil {
		return ManifestAdmissionPlan{}, err
	}

	if err := packageOverlay.EnsureAll(packages); err != nil {
		return ManifestAdmissionPlan{}, err
	}

	sourceOverlay := NewPackageSourceRegistry()
	if err := sourceOverlay.EnsureAll(existingSources); err != nil {
		return ManifestAdmissionPlan{}, err
	}

	if err := sourceOverlay.EnsureAll(sources); err != nil {
		return ManifestAdmissionPlan{}, err
	}

	resolved, err := manifest.ResolveDependencies(m, packageOverlay)
	if err != nil {
		return ManifestAdmissionPlan{}, err
	}

	buildPlan, err := manifest.BuildPlanForManifest(resolved)
	if err != nil {
		return ManifestAdmissionPlan{}, err
	}

	return ManifestAdmissionPlan{
		buildPlan: buildPlan,
		packages:  packages,
		sources:   sources,
	}, nil
}

func cloneManifestAdmissionBuildPlan(
	plan manifest.BuildPlan,
) manifest.BuildPlan {
	clone := plan
	if plan.Steps == nil {
		return clone
	}

	clone.Steps = make([]manifest.BuildStep, len(plan.Steps))
	for i, step := range plan.Steps {
		clone.Steps[i] = step
		if step.Dependencies == nil {
			continue
		}

		clone.Steps[i].Dependencies = make(
			[]string,
			len(step.Dependencies),
		)
		copy(clone.Steps[i].Dependencies, step.Dependencies)
	}

	return clone
}

func cloneManifestAdmissionPackages(
	packages []registry.Package,
) []registry.Package {
	if packages == nil {
		return nil
	}

	clone := make([]registry.Package, len(packages))
	copy(clone, packages)

	return clone
}

func cloneManifestAdmissionSources(
	sources []PackageSource,
) []PackageSource {
	if sources == nil {
		return nil
	}

	clone := make([]PackageSource, len(sources))
	copy(clone, sources)

	return clone
}
