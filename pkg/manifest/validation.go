package manifest

import (
	"fmt"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

// Validate validates the manifest contract.
func (m Manifest) Validate() error {
	if m.Version == "" {
		return invalidManifest("manifest.version is required")
	}

	if m.Name == "" {
		return invalidManifest("manifest.name is required")
	}

	seen := make(map[string]struct{}, len(m.Modules))

	for i, module := range m.Modules {
		if module.Name == "" {
			return invalidManifest(
				fmt.Sprintf("manifest.modules[%d].name is required", i),
			)
		}

		if module.Version == "" {
			return invalidManifest(
				fmt.Sprintf("manifest.modules[%d].version is required", i),
			)
		}

		if _, exists := seen[module.Name]; exists {
			return invalidManifest(
				fmt.Sprintf("manifest.modules contains duplicate module %q", module.Name),
			)
		}

		seen[module.Name] = struct{}{}
	}

	return nil
}

func invalidManifest(message string) error {
	return &forgeerrors.Error{
		Code:    forgeerrors.CodeInvalidManifest,
		Message: message,
	}
}
