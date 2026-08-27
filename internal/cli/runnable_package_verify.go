package cli

import (
	"crypto/ed25519"
	"fmt"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

type runnablePackageExpectation struct {
	KeyID           string
	PublicKey       ed25519.PublicKey
	ManifestName    string
	ManifestVersion string
	Entrypoint      compiler.RuntimeEntrypoint
	TargetOS        string
	TargetArch      string
	ImportPath      string
}

func verifyStagedRunnablePackage(
	path string,
	expected runnablePackageExpectation,
) error {
	keyID, err := validateRunnableSigningKeyID(expected.KeyID)
	if err != nil {
		return runnablePackageVerificationError("invalid expected signer key ID", err)
	}
	if len(expected.PublicKey) != ed25519.PublicKeySize {
		return runnablePackageVerificationError("invalid expected Ed25519 public key", nil)
	}
	if strings.TrimSpace(path) == "" || strings.TrimSpace(expected.ManifestName) == "" ||
		strings.TrimSpace(expected.ManifestVersion) == "" ||
		strings.TrimSpace(expected.Entrypoint.Module) == "" ||
		strings.TrimSpace(expected.Entrypoint.Version) == "" ||
		strings.TrimSpace(expected.TargetOS) == "" || strings.TrimSpace(expected.TargetArch) == "" ||
		strings.TrimSpace(expected.ImportPath) == "" {
		return runnablePackageVerificationError("runnable package expectation is incomplete", nil)
	}

	trustStore := compiler.NewTrustStore()
	if err := trustStore.Register(keyID, expected.PublicKey); err != nil {
		return runnablePackageVerificationError("register build-side verification key", err)
	}
	verifier, err := compiler.NewEd25519VerifierWithTrustStore(trustStore)
	if err != nil {
		return runnablePackageVerificationError("create build-side signature verifier", err)
	}
	reader, err := compiler.NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
		compiler.StrictPackageVerificationPolicy(),
		verifier,
		compiler.AlphaRuntimePackageReadLimits(),
	)
	if err != nil {
		return runnablePackageVerificationError("create strict bounded package reader", err)
	}

	result, err := reader.ReadDetailed(path)
	if err != nil {
		return runnablePackageVerificationError("strictly verify staged runnable package", err)
	}
	if result.PackageFormatVersion != 2 || result.BundleSchemaVersion != 2 {
		return runnablePackageVerificationError("staged package is not package format 2 / bundle schema 2", nil)
	}
	if result.VerifiedSignerKeyID != keyID {
		return runnablePackageVerificationError("verified signer key ID does not match expectation", nil)
	}

	bundle := result.Bundle
	if bundle.ManifestName != expected.ManifestName || bundle.ManifestVersion != expected.ManifestVersion {
		return runnablePackageVerificationError("package manifest identity does not match expectation", nil)
	}
	if bundle.Runtime == nil {
		return runnablePackageVerificationError("package runtime descriptor is missing", nil)
	}
	if bundle.Runtime.Kind != compiler.RuntimeKindApplicationExecutable ||
		bundle.Runtime.Entrypoint != expected.Entrypoint ||
		bundle.Runtime.TargetOS != expected.TargetOS ||
		bundle.Runtime.TargetArch != expected.TargetArch {
		return runnablePackageVerificationError("package runtime descriptor does not match expectation", nil)
	}
	if len(bundle.Artifacts) != 1 {
		return runnablePackageVerificationError("package must contain exactly one artifact", nil)
	}

	artifact := bundle.Artifacts[0]
	if artifact.Module != expected.Entrypoint.Module ||
		artifact.Version != expected.Entrypoint.Version || artifact.ImportPath != expected.ImportPath {
		return runnablePackageVerificationError("package artifact does not match expectation", nil)
	}
	payloadKey := expected.Entrypoint.Module + "@" + expected.Entrypoint.Version
	payload, present := result.Payloads[payloadKey]
	if len(result.Payloads) != 1 || !present || len(payload) == 0 {
		return runnablePackageVerificationError("package executable payload is missing or empty", nil)
	}

	return nil
}

func runnablePackageVerificationError(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", compiler.ErrInvalidArtifactPackage, message)
	}

	return fmt.Errorf("%w: %s: %w", compiler.ErrInvalidArtifactPackage, message, cause)
}
