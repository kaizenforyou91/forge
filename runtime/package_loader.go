package runtime

import (
	"fmt"
	goruntime "runtime"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

const (
	runnablePackageFormatVersion = 2
	runnableBundleSchemaVersion  = 2
)

// VerifiedRunnablePackageLoader strictly verifies and host-authorizes Forge
// runnable packages without materializing or executing their payloads.
type VerifiedRunnablePackageLoader struct {
	reader *compiler.ZIPPackageReader
}

// NewVerifiedRunnablePackageLoader creates a loader with fixed strict
// verification and Alpha bounded-read policy. Callers control only trust.
func NewVerifiedRunnablePackageLoader(
	trustStore *compiler.TrustStore,
) (*VerifiedRunnablePackageLoader, error) {
	verifier, err := compiler.NewEd25519VerifierWithTrustStore(trustStore)
	if err != nil {
		return nil, err
	}

	reader, err := compiler.NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
		compiler.StrictPackageVerificationPolicy(),
		verifier,
		compiler.AlphaRuntimePackageReadLimits(),
	)
	if err != nil {
		return nil, err
	}

	return &VerifiedRunnablePackageLoader{reader: reader}, nil
}

// Load strictly verifies a package, validates the Alpha runnable contract,
// authorizes its exact host target, and returns detached executable bytes.
func (l *VerifiedRunnablePackageLoader) Load(
	packagePath string,
) (VerifiedRunnablePackage, error) {
	if l == nil || l.reader == nil {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: loader is nil or incomplete",
			ErrInvalidRunnablePackage,
		)
	}
	if strings.TrimSpace(packagePath) == "" {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: package path is required",
			ErrInvalidRunnablePackage,
		)
	}

	result, err := l.reader.ReadDetailed(packagePath)
	if err != nil {
		return VerifiedRunnablePackage{}, err
	}

	return verifiedRunnablePackageFromReadResult(result)
}

func verifiedRunnablePackageFromReadResult(
	result compiler.PackageReadResult,
) (VerifiedRunnablePackage, error) {
	if result.PackageFormatVersion == 1 && result.BundleSchemaVersion == 1 {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: package format 1 / bundle schema 1",
			ErrPackageNotRunnable,
		)
	}
	if result.PackageFormatVersion != runnablePackageFormatVersion ||
		result.BundleSchemaVersion != runnableBundleSchemaVersion {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: unexpected package format %d / bundle schema %d",
			ErrInvalidRunnablePackage,
			result.PackageFormatVersion,
			result.BundleSchemaVersion,
		)
	}
	if strings.TrimSpace(result.VerifiedSignerKeyID) == "" {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: strict read returned no verified signer evidence",
			ErrInvalidRunnablePackage,
		)
	}

	bundle := result.Bundle
	if bundle.Runtime == nil {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: runtime descriptor is required",
			ErrInvalidRunnablePackage,
		)
	}
	if bundle.Runtime.Kind != compiler.RuntimeKindApplicationExecutable {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: unsupported runtime kind %q",
			ErrInvalidRunnablePackage,
			bundle.Runtime.Kind,
		)
	}
	if len(bundle.Artifacts) != 1 {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: expected exactly one artifact, got %d",
			ErrInvalidRunnablePackage,
			len(bundle.Artifacts),
		)
	}

	entrypoint := bundle.Runtime.Entrypoint
	if strings.TrimSpace(entrypoint.Module) == "" ||
		strings.TrimSpace(entrypoint.Version) == "" {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: runtime entrypoint identity is incomplete",
			ErrInvalidRunnablePackage,
		)
	}

	artifact := bundle.Artifacts[0]
	if artifact.Module != entrypoint.Module || artifact.Version != entrypoint.Version {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: entrypoint %s@%s does not match artifact %s@%s",
			ErrInvalidRunnablePackage,
			entrypoint.Module,
			entrypoint.Version,
			artifact.Module,
			artifact.Version,
		)
	}
	if strings.TrimSpace(artifact.ImportPath) == "" {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: entrypoint import path is required",
			ErrInvalidRunnablePackage,
		)
	}

	if bundle.Runtime.TargetOS != goruntime.GOOS ||
		bundle.Runtime.TargetArch != goruntime.GOARCH {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: package targets %s/%s, host is %s/%s",
			ErrUnsupportedRuntimePlatform,
			bundle.Runtime.TargetOS,
			bundle.Runtime.TargetArch,
			goruntime.GOOS,
			goruntime.GOARCH,
		)
	}

	payloadKey := entrypoint.Module + "@" + entrypoint.Version
	executable, ok := result.Payloads[payloadKey]
	if !ok || len(executable) == 0 {
		return VerifiedRunnablePackage{}, fmt.Errorf(
			"%w: executable payload %q is missing or empty",
			ErrInvalidRunnablePackage,
			payloadKey,
		)
	}

	return VerifiedRunnablePackage{
		packageFormatVersion: result.PackageFormatVersion,
		bundleSchemaVersion:  result.BundleSchemaVersion,
		manifestName:         bundle.ManifestName,
		manifestVersion:      bundle.ManifestVersion,
		entrypoint:           entrypoint,
		importPath:           artifact.ImportPath,
		targetOS:             bundle.Runtime.TargetOS,
		targetArch:           bundle.Runtime.TargetArch,
		signerKeyID:          result.VerifiedSignerKeyID,
		executable:           append([]byte(nil), executable...),
	}, nil
}
