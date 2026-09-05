package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var trustedKeyPath string
	var keyIDValue string

	cmd := &cobra.Command{
		Use:   "inspect <package.zip>",
		Short: "Inspect a Forge package without executing it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := ApplicationFromContext(cmd.Context()); err != nil {
				return err
			}

			trustedKeySet := cmd.Flags().Changed("trusted-key")
			keyIDSet := cmd.Flags().Changed("key-id")
			if trustedKeySet != keyIDSet {
				return runTrustedKeyError(
					"--trusted-key and --key-id must be supplied together",
					nil,
				)
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get inspect working directory: %w", err)
			}
			packagePath, err := resolveLocalPackagePath(cwd, args[0])
			if err != nil {
				return err
			}

			var result compiler.PackageInspectionResult
			if trustedKeySet {
				keyID, err := validateTrustedKeyID(keyIDValue)
				if err != nil {
					return err
				}
				publicKey, err := loadTrustedPublicKey(trustedKeyPath)
				if err != nil {
					return err
				}

				trustStore := compiler.NewTrustStore()
				if err := trustStore.Register(keyID, publicKey); err != nil {
					return err
				}
				verifier, err := compiler.NewEd25519VerifierWithTrustStore(trustStore)
				if err != nil {
					return err
				}
				result, err = compiler.InspectPackageWithVerifier(packagePath, verifier)
				if err != nil {
					return err
				}
			} else {
				result, err = compiler.InspectPackage(packagePath)
				if err != nil {
					return err
				}
			}

			report, err := formatPackageInspection(packagePath, result)
			if err != nil {
				return err
			}
			if _, err := io.Copy(cmd.OutOrStdout(), bytes.NewReader(report)); err != nil {
				return fmt.Errorf("write package inspection report: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(
		&trustedKeyPath,
		"trusted-key",
		"",
		"Ed25519 public key in X.509 PKIX PEM format",
	)
	cmd.Flags().StringVar(
		&keyIDValue,
		"key-id",
		"",
		"exact trusted signing key ID",
	)

	return cmd
}

func formatPackageInspection(
	packagePath string,
	result compiler.PackageInspectionResult,
) ([]byte, error) {
	var report bytes.Buffer

	writeInspectionField(&report, "Package", packagePath)
	writeInspectionField(&report, "Format", strconv.Itoa(result.PackageFormatVersion))
	writeInspectionField(&report, "Bundle schema", strconv.Itoa(result.BundleSchemaVersion))
	writeInspectionField(
		&report,
		"Manifest",
		result.Bundle.ManifestName+"@"+result.Bundle.ManifestVersion,
	)

	switch {
	case result.PackageFormatVersion == 1 && result.BundleSchemaVersion == 1:
		writeInspectionField(&report, "Type", "non-runnable")
		writeInspectionField(&report, "Runtime", "none")
	case result.PackageFormatVersion == 2 && result.BundleSchemaVersion == 2:
		if result.Bundle.Runtime == nil {
			return nil, fmt.Errorf(
				"%w: inspected runnable package has no runtime metadata",
				compiler.ErrInvalidArtifactPackage,
			)
		}
		writeInspectionField(&report, "Type", "runnable")
		writeInspectionField(&report, "Runtime", result.Bundle.Runtime.Kind)
		writeInspectionField(
			&report,
			"Target",
			result.Bundle.Runtime.TargetOS+"/"+result.Bundle.Runtime.TargetArch,
		)
		writeInspectionField(
			&report,
			"Entrypoint",
			result.Bundle.Runtime.Entrypoint.Module+"@"+
				result.Bundle.Runtime.Entrypoint.Version,
		)
	default:
		return nil, fmt.Errorf(
			"%w: unsupported inspected package format %d / bundle schema %d",
			compiler.ErrUnsupportedPackageFormat,
			result.PackageFormatVersion,
			result.BundleSchemaVersion,
		)
	}

	writeInspectionField(&report, "Integrity", "verified")
	switch result.SignatureState {
	case compiler.PackageSignatureUnsigned:
		writeInspectionField(&report, "Signature", "unsigned")
	case compiler.PackageSignatureSignedUnverified:
		writeInspectionField(&report, "Signature", "present, trust not verified")
		writeInspectionField(
			&report,
			"Declared KeyID (unverified)",
			result.DeclaredKeyID,
		)
	case compiler.PackageSignatureSignedTrusted:
		writeInspectionField(&report, "Signature", "trusted")
		writeInspectionField(&report, "Verified signer", result.VerifiedSignerKeyID)
	default:
		return nil, fmt.Errorf(
			"%w: unknown package signature state %q",
			compiler.ErrInvalidPackageSignature,
			result.SignatureState,
		)
	}

	writeInspectionField(&report, "Artifacts", strconv.Itoa(len(result.Bundle.Artifacts)))
	for _, artifact := range result.Bundle.Artifacts {
		_, _ = fmt.Fprintf(
			&report,
			"  - %s@%s - %s\n",
			artifact.Module,
			artifact.Version,
			artifact.ImportPath,
		)
	}

	return report.Bytes(), nil
}

func writeInspectionField(report *bytes.Buffer, name string, value string) {
	_, _ = fmt.Fprintf(report, "%s: %s\n", name, value)
}
