package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/kaizenforyou91/forge/pkg/compiler"
	forgeruntime "github.com/kaizenforyou91/forge/runtime"
	"github.com/spf13/cobra"
)

const runProcessOutputLimit = 1 * 1024 * 1024

func newRunCmd() *cobra.Command {
	var trustedKeyPath string
	var keyIDValue string

	cmd := &cobra.Command{
		Use:   "run <package.zip>",
		Short: "Execute a trusted runnable package",
		Long: "Execute a strictly verified, explicitly trusted runnable package.\n\n" +
			"Warning: trusted packages execute code directly; Forge provides no sandbox, " +
			"process-tree containment, or resource isolation.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := ApplicationFromContext(cmd.Context()); err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get run working directory: %w", err)
			}
			packagePath, err := resolveRunPackagePath(cwd, args[0])
			if err != nil {
				return err
			}

			keyID, err := validateRunTrustKeyID(keyIDValue)
			if err != nil {
				return err
			}
			publicKey, err := loadRunTrustedPublicKey(trustedKeyPath)
			if err != nil {
				return err
			}

			trustStore := compiler.NewTrustStore()
			if err := trustStore.Register(keyID, publicKey); err != nil {
				return err
			}
			loader, err := forgeruntime.NewVerifiedRunnablePackageLoader(trustStore)
			if err != nil {
				return err
			}
			verifiedPackage, err := loader.Load(packagePath)
			if err != nil {
				return err
			}

			materialized, err := forgeruntime.NewSecureExecutableMaterializer().Materialize(verifiedPackage)
			if err != nil {
				return err
			}
			return executeRunMaterialized(cmd, materialized)
		},
	}

	cmd.Flags().StringVar(&trustedKeyPath, "trusted-key", "", "Ed25519 public key in X.509 PKIX PEM format")
	cmd.Flags().StringVar(&keyIDValue, "key-id", "", "exact trusted signing key ID")
	_ = cmd.MarkFlagRequired("trusted-key")
	_ = cmd.MarkFlagRequired("key-id")

	return cmd
}

func executeRunMaterialized(cmd *cobra.Command, materialized *forgeruntime.MaterializedExecutable) error {
	running, startErr := forgeruntime.NewProcessRunner().Start(cmd.Context(), materialized)
	if startErr != nil {
		return errors.Join(startErr, materialized.Close())
	}

	result, waitErr := running.Wait()
	outputErr := writeRunProcessResult(cmd, result)
	cleanupErr := materialized.Close()
	if outputErr != nil || cleanupErr != nil {
		return errors.Join(waitErr, outputErr, cleanupErr)
	}
	if result.Canceled && errors.Is(waitErr, context.Canceled) {
		return context.Canceled
	}
	if waitErr != nil {
		return waitErr
	}
	if result.Terminated {
		return fmt.Errorf("child process terminated without a CLI termination request")
	}
	if result.ExitCode == 0 {
		return nil
	}
	return newExitStatusError(result.ExitCode, nil)
}

func writeRunProcessResult(cmd *cobra.Command, result forgeruntime.ProcessResult) error {
	var writeErrors []error
	if _, err := cmd.OutOrStdout().Write(result.Stdout); err != nil {
		writeErrors = append(writeErrors, fmt.Errorf("write child stdout: %w", err))
	}
	if _, err := cmd.ErrOrStderr().Write(result.Stderr); err != nil {
		writeErrors = append(writeErrors, fmt.Errorf("write child stderr: %w", err))
	}
	if result.StdoutTruncated {
		if _, err := io.WriteString(cmd.ErrOrStderr(), "forge: child stdout truncated after 1048576 bytes\n"); err != nil {
			writeErrors = append(writeErrors, fmt.Errorf("write child stdout truncation warning: %w", err))
		}
	}
	if result.StderrTruncated {
		if _, err := io.WriteString(cmd.ErrOrStderr(), "forge: child stderr truncated after 1048576 bytes\n"); err != nil {
			writeErrors = append(writeErrors, fmt.Errorf("write child stderr truncation warning: %w", err))
		}
	}
	return errors.Join(writeErrors...)
}
