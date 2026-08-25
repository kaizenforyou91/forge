package manifest

import (
	"fmt"
	"strings"

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

		seenDependencies := make(map[string]struct{}, len(module.Dependencies))

		for dependencyIndex, dependency := range module.Dependencies {
			if dependency.Name == "" {
				return invalidManifest(
					fmt.Sprintf(
						"manifest.modules[%d].dependencies[%d].name is required",
						i,
						dependencyIndex,
					),
				)
			}

			if dependency.Version == "" {
				return invalidManifest(
					fmt.Sprintf(
						"manifest.modules[%d].dependencies[%d].version is required",
						i,
						dependencyIndex,
					),
				)
			}

			if dependency.Name == module.Name &&
				dependency.Version == module.Version {
				return invalidManifest(
					fmt.Sprintf(
						"manifest.modules[%d] cannot depend on itself %q@%q",
						i,
						dependency.Name,
						dependency.Version,
					),
				)
			}

			key := dependency.Name + "@" + dependency.Version

			if _, exists := seenDependencies[key]; exists {
				return invalidManifest(
					fmt.Sprintf(
						"manifest.modules[%d] contains duplicate dependency %q",
						i,
						key,
					),
				)
			}

			seenDependencies[key] = struct{}{}
		}

		if _, exists := seen[module.Name]; exists {
			return invalidManifest(
				fmt.Sprintf("manifest.modules contains duplicate module %q", module.Name),
			)
		}

		seen[module.Name] = struct{}{}
	}

	if m.Entrypoint == nil {
		return nil
	}

	entrypointModule := m.Entrypoint.Module
	entrypointVersion := m.Entrypoint.Version

	if strings.TrimSpace(entrypointModule) == "" {
		return invalidManifest("manifest.entrypoint.module is required")
	}

	if strings.TrimSpace(entrypointVersion) == "" {
		return invalidManifest("manifest.entrypoint.version is required")
	}

	if strings.TrimSpace(entrypointModule) != entrypointModule {
		return invalidManifest(
			"manifest.entrypoint.module must not contain surrounding whitespace",
		)
	}

	if strings.TrimSpace(entrypointVersion) != entrypointVersion {
		return invalidManifest(
			"manifest.entrypoint.version must not contain surrounding whitespace",
		)
	}

	matches := 0
	for _, module := range m.Modules {
		if module.Name == entrypointModule &&
			module.Version == entrypointVersion {
			matches++
		}
	}

	if matches != 1 {
		return invalidManifest(
			fmt.Sprintf(
				"manifest.entrypoint references unknown module %q@%q",
				entrypointModule,
				entrypointVersion,
			),
		)
	}

	return nil
}

func invalidManifest(message string) error {
	return &forgeerrors.Error{
		Code:    forgeerrors.CodeInvalidManifest,
		Message: message,
	}
}
