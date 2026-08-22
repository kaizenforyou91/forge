package compiler

import (
	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

// AdmitManifest prepares and commits manifest package and source candidates.
//
// Preflight uses independent registry snapshots. Source candidates are then
// revalidated against live state before package candidates are committed.
func AdmitManifest(
	m manifest.Manifest,
	packages *registry.Registry,
	sources *PackageSourceRegistry,
) (ManifestAdmissionPlan, error) {
	if packages == nil {
		return ManifestAdmissionPlan{}, registry.ErrInvalidPackage
	}

	if sources == nil {
		return ManifestAdmissionPlan{}, ErrInvalidPackageSource
	}

	admission, err := PrepareManifestAdmission(
		m,
		packages.List(),
		sources.List(),
	)
	if err != nil {
		return ManifestAdmissionPlan{}, err
	}

	if err := commitManifestAdmission(
		admission,
		packages,
		sources,
	); err != nil {
		return ManifestAdmissionPlan{}, err
	}

	return admission, nil
}

func commitManifestAdmission(
	admission ManifestAdmissionPlan,
	packages *registry.Registry,
	sources *PackageSourceRegistry,
) error {
	if packages == nil {
		return registry.ErrInvalidPackage
	}

	if sources == nil {
		return ErrInvalidPackageSource
	}

	if err := sources.EnsureAll(admission.Sources()); err != nil {
		return err
	}

	return packages.EnsureAll(admission.Packages())
}
