package cli

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/spf13/cobra"
)

func newBuildRunnableCmd() *cobra.Command {
	var signingKeyPath string
	var keyIDValue string
	var requestedOutput string

	cmd := &cobra.Command{
		Use:   "build-runnable <manifest>",
		Short: "Build a signed runnable package from a Forge manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			application, err := ApplicationFromContext(cmd.Context())
			if err != nil {
				return err
			}

			m, err := loadManifestFile(args[0])
			if err != nil {
				return err
			}

			registryInstance, err := resolveRegistry(application)
			if err != nil {
				return err
			}
			sourceRegistry, err := resolveSourceRegistry(application)
			if err != nil {
				return err
			}
			runner, err := resolveOSCommandRunner(application)
			if err != nil {
				return err
			}

			admission, err := compiler.PrepareManifestAdmission(
				m,
				registryInstance.List(),
				sourceRegistry.List(),
			)
			if err != nil {
				return err
			}

			entrypoint, present := admission.ApplicationEntrypoint()
			if !present {
				return fmt.Errorf(
					"%w: manifest has no application entrypoint",
					compiler.ErrInvalidApplicationEntrypoint,
				)
			}
			admittedSource, err := runnableBuildAdmittedSource(
				admission,
				entrypoint.Module,
				entrypoint.Version,
			)
			if err != nil {
				return err
			}

			if err := cmd.Context().Err(); err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get runnable build working directory: %w", err)
			}
			cwd, err = filepath.Abs(cwd)
			if err != nil {
				return fmt.Errorf("resolve runnable build working directory: %w", err)
			}
			cwd = filepath.Clean(cwd)

			finalPath, err := resolveRunnableOutputPath(
				cwd,
				requestedOutput,
				m.Name,
				m.Version,
				runtime.GOOS,
				runtime.GOARCH,
			)
			if err != nil {
				return err
			}

			keyID, err := validateRunnableSigningKeyID(keyIDValue)
			if err != nil {
				return err
			}
			privateKey, err := loadRunnableSigningPrivateKey(signingKeyPath)
			if err != nil {
				return err
			}
			defer clearBytes(privateKey)

			publicKey := append(
				ed25519.PublicKey(nil),
				privateKey.Public().(ed25519.PublicKey)...,
			)
			signer, err := compiler.NewEd25519Signer(keyID, privateKey)
			if err != nil {
				return err
			}
			packager := compiler.NewZIPPackagerWithSigner(signer)
			builder, err := compiler.NewGoApplicationExecutableBuilder(runner)
			if err != nil {
				return err
			}
			runnableCompiler, err := compiler.NewRunnableManifestCompiler(
				builder,
				packager,
			)
			if err != nil {
				return err
			}

			stage, err := prepareRunnableOutputStage(finalPath)
			if err != nil {
				return err
			}
			cleanupFailure := func(primary error) error {
				cleanupErr := stage.Cleanup()
				if cleanupErr == nil {
					return primary
				}
				if primary == nil {
					return cleanupErr
				}
				return errors.Join(primary, cleanupErr)
			}

			if err := cmd.Context().Err(); err != nil {
				return cleanupFailure(err)
			}
			if err := runnableCompiler.Compile(
				cmd.Context(),
				compiler.RunnableManifestRequest{
					Admission:        admission,
					WorkingDirectory: cwd,
					OutputPath:       stage.packagePath,
				},
			); err != nil {
				return cleanupFailure(err)
			}
			if err := cmd.Context().Err(); err != nil {
				return cleanupFailure(err)
			}

			expected := runnablePackageExpectation{
				KeyID:           keyID,
				PublicKey:       publicKey,
				ManifestName:    m.Name,
				ManifestVersion: m.Version,
				Entrypoint: compiler.RuntimeEntrypoint{
					Module:  entrypoint.Module,
					Version: entrypoint.Version,
				},
				TargetOS:   runtime.GOOS,
				TargetArch: runtime.GOARCH,
				ImportPath: admittedSource.ImportPath,
			}
			if err := verifyStagedRunnablePackage(stage.packagePath, expected); err != nil {
				return cleanupFailure(err)
			}
			if err := cmd.Context().Err(); err != nil {
				return cleanupFailure(err)
			}
			if err := stage.Publish(); err != nil {
				return cleanupFailure(err)
			}

			if err := stage.Cleanup(); err != nil {
				return fmt.Errorf(
					"runnable package was published at %q but staging cleanup failed: %w",
					finalPath,
					err,
				)
			}

			if _, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"Runnable package created: %s (format=2, bundle=2, entrypoint=%s@%s, target=%s/%s, signer=%s)\n",
				finalPath,
				entrypoint.Module,
				entrypoint.Version,
				runtime.GOOS,
				runtime.GOARCH,
				keyID,
			); err != nil {
				return fmt.Errorf(
					"runnable package was published at %q but success output failed: %w",
					finalPath,
					err,
				)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(
		&signingKeyPath,
		"signing-key",
		"",
		"Path to an unencrypted PKCS#8 PEM Ed25519 private key",
	)
	cmd.Flags().StringVar(
		&keyIDValue,
		"key-id",
		"",
		"Required signer key ID",
	)
	cmd.Flags().StringVar(
		&requestedOutput,
		"output",
		"",
		"Output runnable package path",
	)

	return cmd
}

func resolveOSCommandRunner(application *app.App) (*compiler.OSCommandRunner, error) {
	if application == nil {
		return nil, fmt.Errorf("application is nil")
	}

	var runner *compiler.OSCommandRunner
	if err := application.Container().Resolve(&runner); err != nil {
		return nil, err
	}
	if runner == nil {
		return nil, fmt.Errorf("compiler OS command runner is nil")
	}

	return runner, nil
}

func runnableBuildAdmittedSource(
	admission compiler.ManifestAdmissionPlan,
	module,
	version string,
) (compiler.PackageSource, error) {
	var selected compiler.PackageSource
	matches := 0
	for _, source := range admission.Sources() {
		if source.Name != module || source.Version != version {
			continue
		}
		selected = source
		matches++
	}

	if matches == 0 {
		return compiler.PackageSource{}, fmt.Errorf(
			"%w: admitted source for %s@%s: %w",
			compiler.ErrInvalidApplicationEntrypoint,
			module,
			version,
			compiler.ErrPackageSourceNotFound,
		)
	}
	if matches != 1 || strings.TrimSpace(selected.ImportPath) == "" {
		return compiler.PackageSource{}, fmt.Errorf(
			"%w: admitted source for %s@%s is invalid: %w",
			compiler.ErrInvalidApplicationEntrypoint,
			module,
			version,
			compiler.ErrInvalidPackageSource,
		)
	}

	return selected, nil
}
