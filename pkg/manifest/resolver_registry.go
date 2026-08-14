package manifest

import (
	"fmt"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

// ResolveFromRegistry resolves manifest modules against a package registry.
//
// Resolution uses exact package name and version matching.
// The registry remains responsible for package identity and storage,
// while the manifest package remains responsible for resolution semantics.
func ResolveFromRegistry(m Manifest, packages *registry.Registry) (Manifest, error) {
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}

	if packages == nil {
		return Manifest{}, &forgeerrors.Error{
			Code:    forgeerrors.CodeInternal,
			Message: "package registry is nil",
		}
	}

	resolved := m
	resolved.Modules = make([]Module, 0, len(m.Modules))

	for _, requested := range m.Modules {
		pkg, err := packages.Get(requested.Name, requested.Version)
		if err != nil {
			if err == registry.ErrPackageNotFound {
				return Manifest{}, &forgeerrors.Error{
					Code: forgeerrors.CodeNotFound,
					Message: fmt.Sprintf(
						"manifest module %q version %q not found",
						requested.Name,
						requested.Version,
					),
					Err: err,
				}
			}

			return Manifest{}, err
		}

		resolved.Modules = append(resolved.Modules, Module{
			Name:    pkg.Name,
			Version: pkg.Version,
		})
	}

	return resolved, nil
}
