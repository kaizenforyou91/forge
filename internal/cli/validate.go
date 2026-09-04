package cli

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/spf13/cobra"
)

type manifestValidationProfile string

const (
	manifestValidationProfileStructural manifestValidationProfile = "structural"
	manifestValidationProfileBuild      manifestValidationProfile = "build"
	manifestValidationProfileRunnable   manifestValidationProfile = "runnable"
)

func newValidateCmd() *cobra.Command {
	profileValue := string(manifestValidationProfileBuild)

	cmd := &cobra.Command{
		Use:   "validate <manifest>",
		Short: "Validate a Forge manifest without building it",
		Long: `Validate a Forge manifest without mutating registries or producing output artifacts.

Profiles:
  structural  Validate strict decoding and manifest-domain rules.
  build       Also validate non-mutating compiler admission (default).
  runnable    Also require an admitted application entrypoint and source.

Validation does not prove compilation, signing, publication, or runtime success.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manifestPath := args[0]
			profile, err := parseManifestValidationProfile(profileValue)
			if err != nil {
				return wrapManifestValidationError(manifestPath, profileValue, err)
			}

			m, err := loadManifestFile(manifestPath)
			if err != nil {
				return wrapManifestValidationError(manifestPath, profileValue, err)
			}

			if err := validateManifestForProfile(cmd, m, profile); err != nil {
				return wrapManifestValidationError(manifestPath, profileValue, err)
			}

			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"Manifest valid: %s (profile=%s)\n",
				manifestPath,
				profile,
			); err != nil {
				return wrapManifestValidationError(
					manifestPath,
					profileValue,
					fmt.Errorf("write success output: %w", err),
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(
		&profileValue,
		"profile",
		string(manifestValidationProfileBuild),
		"Validation profile: structural, build, or runnable",
	)

	return cmd
}

func parseManifestValidationProfile(value string) (manifestValidationProfile, error) {
	profile := manifestValidationProfile(value)
	switch profile {
	case manifestValidationProfileStructural,
		manifestValidationProfileBuild,
		manifestValidationProfileRunnable:
		return profile, nil
	default:
		return "", fmt.Errorf("invalid validation profile %q", value)
	}
}

func validateManifestForProfile(
	cmd *cobra.Command,
	m manifest.Manifest,
	profile manifestValidationProfile,
) error {
	if profile != manifestValidationProfileStructural &&
		profile != manifestValidationProfileBuild &&
		profile != manifestValidationProfileRunnable {
		return fmt.Errorf("invalid validation profile %q", profile)
	}
	if err := m.Validate(); err != nil {
		return err
	}
	if profile == manifestValidationProfileStructural {
		return nil
	}

	application, err := ApplicationFromContext(cmd.Context())
	if err != nil {
		return err
	}
	admission, err := prepareManifestValidationAdmission(application, m)
	if err != nil {
		return err
	}
	if profile == manifestValidationProfileBuild {
		return nil
	}

	_, _, err = runnableBuildAdmittedEntrypoint(admission)
	return err
}

func prepareManifestValidationAdmission(
	application *app.App,
	m manifest.Manifest,
) (compiler.ManifestAdmissionPlan, error) {
	registryInstance, err := resolveRegistry(application)
	if err != nil {
		return compiler.ManifestAdmissionPlan{}, err
	}
	sourceRegistry, err := resolveSourceRegistry(application)
	if err != nil {
		return compiler.ManifestAdmissionPlan{}, err
	}

	return compiler.PrepareManifestAdmission(
		m,
		registryInstance.List(),
		sourceRegistry.List(),
	)
}

func wrapManifestValidationError(
	manifestPath,
	profile string,
	err error,
) error {
	return fmt.Errorf(
		"validate manifest %q (profile=%s): %w",
		manifestPath,
		profile,
		err,
	)
}
