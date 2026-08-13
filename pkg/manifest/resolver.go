package manifest

import (
	"fmt"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

// Resolve binds manifest module declarations to available modules.
//
// Resolution uses exact module name and version matching.
// Discovery, remote lookup, and version constraint solving are outside
// the scope of the manifest foundation.
func Resolve(m Manifest, available []Module) (Manifest, error) {
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}

	index := make(map[string]Module, len(available))

	for _, module := range available {
		key := module.Name + "@" + module.Version

		if _, exists := index[key]; !exists {
			index[key] = module
		}
	}

	resolved := m
	resolved.Modules = make([]Module, 0, len(m.Modules))

	for _, requested := range m.Modules {
		key := requested.Name + "@" + requested.Version

		module, ok := index[key]
		if !ok {
			return Manifest{}, &forgeerrors.Error{
				Code: forgeerrors.CodeNotFound,
				Message: fmt.Sprintf(
					"manifest module %q version %q not found",
					requested.Name,
					requested.Version,
				),
			}
		}

		resolved.Modules = append(resolved.Modules, module)
	}

	return resolved, nil
}
