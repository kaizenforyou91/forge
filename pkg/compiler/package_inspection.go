package compiler

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

// InspectPackage performs bounded package inspection with mandatory integrity
// verification. A present signature is verified against its embedded public
// key, but the key is not treated as trusted.
func InspectPackage(path string) (PackageInspectionResult, error) {
	return inspectPackageWithVerifierAndLimits(
		path,
		nil,
		false,
		AlphaPackageReadLimits(),
	)
}

// InspectPackageWithVerifier performs bounded package inspection and requires
// a present signature to verify with the supplied trusted verifier.
func InspectPackageWithVerifier(
	path string,
	verifier PackageVerifier,
) (PackageInspectionResult, error) {
	if verifier == nil {
		return PackageInspectionResult{}, ErrPackageVerifierRequired
	}

	return inspectPackageWithVerifierAndLimits(
		path,
		verifier,
		true,
		AlphaPackageReadLimits(),
	)
}

func inspectPackageWithVerifierAndLimits(
	path string,
	verifier PackageVerifier,
	requireTrustedSignature bool,
	limits PackageReadLimits,
) (PackageInspectionResult, error) {
	reader, err := NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
		DefaultPackageVerificationPolicy(),
		nil,
		limits,
	)
	if err != nil {
		return PackageInspectionResult{}, err
	}

	return inspectPackageWithReader(
		path,
		reader,
		verifier,
		requireTrustedSignature,
	)
}

func inspectPackageWithReader(
	path string,
	reader *ZIPPackageReader,
	verifier PackageVerifier,
	requireTrustedSignature bool,
) (PackageInspectionResult, error) {
	evidence, err := reader.readDetailed(path)
	if err != nil {
		return PackageInspectionResult{}, err
	}

	result := PackageInspectionResult{
		PackageFormatVersion: evidence.PackageFormatVersion,
		BundleSchemaVersion:  evidence.BundleSchemaVersion,
		Bundle:               evidence.Bundle,
		SignatureState:       PackageSignatureUnsigned,
	}

	if !evidence.SignaturePresent {
		if requireTrustedSignature {
			return PackageInspectionResult{}, ErrMissingPackageSignature
		}

		return result, nil
	}

	if err := verifyEmbeddedPackageSignature(
		evidence.IntegrityData,
		evidence.Signature,
	); err != nil {
		return PackageInspectionResult{}, err
	}

	result.SignatureState = PackageSignatureSignedUnverified
	result.DeclaredKeyID = evidence.Signature.KeyID

	if !requireTrustedSignature {
		return result, nil
	}
	if verifier == nil {
		return PackageInspectionResult{}, ErrPackageVerifierRequired
	}
	if err := verifier.Verify(evidence.IntegrityData, evidence.Signature); err != nil {
		return PackageInspectionResult{}, err
	}

	result.SignatureState = PackageSignatureSignedTrusted
	result.VerifiedSignerKeyID = evidence.Signature.KeyID
	return result, nil
}

func verifyEmbeddedPackageSignature(
	payload []byte,
	signature PackageSignature,
) error {
	if err := signature.Validate(); err != nil {
		return err
	}

	publicKey, err := decodeEd25519PublicKey(signature.PublicKey)
	if err != nil {
		return err
	}
	rawSignature, err := base64.StdEncoding.DecodeString(signature.Signature)
	if err != nil || len(rawSignature) != ed25519.SignatureSize {
		return fmt.Errorf(
			"%w: invalid Ed25519 signature",
			ErrInvalidPackageSignature,
		)
	}
	if !ed25519.Verify(publicKey, payload, rawSignature) {
		return ErrSignatureMismatch
	}

	return nil
}
