package manifest

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

// Validate validates the manifest contract.
func (m Manifest) Validate() error {
	if err := validateManifestIdentity(
		"manifest.version",
		m.Version,
	); err != nil {
		return err
	}

	if err := validateManifestIdentity("manifest.name", m.Name); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(m.Modules))

	for i, module := range m.Modules {
		if err := validateManifestIdentity(
			fmt.Sprintf("manifest.modules[%d].name", i),
			module.Name,
		); err != nil {
			return err
		}

		if err := validateManifestIdentity(
			fmt.Sprintf("manifest.modules[%d].version", i),
			module.Version,
		); err != nil {
			return err
		}

		seenDependencies := make(map[string]struct{}, len(module.Dependencies))

		for dependencyIndex, dependency := range module.Dependencies {
			if err := validateManifestIdentity(
				fmt.Sprintf(
					"manifest.modules[%d].dependencies[%d].name",
					i,
					dependencyIndex,
				),
				dependency.Name,
			); err != nil {
				return err
			}

			if err := validateManifestIdentity(
				fmt.Sprintf(
					"manifest.modules[%d].dependencies[%d].version",
					i,
					dependencyIndex,
				),
				dependency.Version,
			); err != nil {
				return err
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

	if err := validateManifestIdentity(
		"manifest.entrypoint.module",
		entrypointModule,
	); err != nil {
		return err
	}

	if err := validateManifestIdentity(
		"manifest.entrypoint.version",
		entrypointVersion,
	); err != nil {
		return err
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

func validateManifestIdentity(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalidManifest(field + " is required")
	}
	if !utf8.ValidString(value) {
		return invalidManifest(field + " must be valid UTF-8")
	}
	if strings.TrimFunc(value, unicode.IsSpace) != value {
		return invalidManifest(field + " must not contain surrounding whitespace")
	}

	for _, r := range value {
		if r <= '\x1f' || r == '\x7f' {
			return invalidManifest(field + " must not contain ASCII control characters")
		}
		if r == '@' {
			return invalidManifest(field + " must not contain '@'")
		}
	}

	return nil
}

func invalidManifest(message string) error {
	return &forgeerrors.Error{
		Code:    forgeerrors.CodeInvalidManifest,
		Message: message,
	}
}
