package compiler

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// Ed25519Verifier verifies trusted Forge package signatures.
type Ed25519Verifier struct {
	trustStore *TrustStore
}

// NewEd25519Verifier creates an Ed25519 verifier with an empty trust store.
func NewEd25519Verifier() *Ed25519Verifier {
	return &Ed25519Verifier{
		trustStore: NewTrustStore(),
	}
}

// NewEd25519VerifierWithTrustStore creates an Ed25519 verifier
// backed by the supplied trust store.
func NewEd25519VerifierWithTrustStore(
	trustStore *TrustStore,
) (*Ed25519Verifier, error) {
	if trustStore == nil {
		return nil, ErrNilTrustStore
	}

	return &Ed25519Verifier{
		trustStore: trustStore,
	}, nil
}

// TrustKey adds a trusted public key.
//
// This method is preserved for backward compatibility.
// New code should prefer registering keys directly in TrustStore.
func (v *Ed25519Verifier) TrustKey(
	keyID string,
	publicKey ed25519.PublicKey,
) error {
	if v == nil {
		return ErrInvalidPackageSignature
	}

	if v.trustStore == nil {
		v.trustStore = NewTrustStore()
	}

	keyID = strings.TrimSpace(keyID)

	if keyID == "" {
		return fmt.Errorf(
			"%w: key ID is required",
			ErrInvalidPackageSignature,
		)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf(
			"%w: invalid Ed25519 public key",
			ErrInvalidPackageSignature,
		)
	}

	if err := v.trustStore.Register(
		keyID,
		publicKey,
	); err != nil {
		if err == ErrDuplicateTrustKey {
			return fmt.Errorf(
				"%w: key %q is already trusted",
				ErrInvalidPackageSignature,
				keyID,
			)
		}

		return err
	}

	return nil
}

// TrustStore returns the trust store used by the verifier.
func (v *Ed25519Verifier) TrustStore() *TrustStore {
	if v == nil {
		return nil
	}

	return v.trustStore
}

// Verify verifies the signature and ensures the signing key is trusted.
func (v *Ed25519Verifier) Verify(
	payload []byte,
	signature PackageSignature,
) error {
	if v == nil || v.trustStore == nil {
		return ErrUntrustedPackageKey
	}

	if err := signature.Validate(); err != nil {
		return err
	}

	publicKey, err := base64.StdEncoding.DecodeString(
		signature.PublicKey,
	)
	if err != nil {
		return fmt.Errorf(
			"%w: decode public key: %v",
			ErrInvalidPackageSignature,
			err,
		)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf(
			"%w: invalid Ed25519 public key",
			ErrInvalidPackageSignature,
		)
	}

	trustedKey, err := v.trustStore.Get(signature.KeyID)
	if err != nil {
		if err == ErrTrustKeyNotFound {
			return fmt.Errorf(
				"%w: key %q",
				ErrUntrustedPackageKey,
				signature.KeyID,
			)
		}

		return err
	}

	if !trustedKey.PublicKey.Equal(
		ed25519.PublicKey(publicKey),
	) {
		return fmt.Errorf(
			"%w: trusted key mismatch for %q",
			ErrUntrustedPackageKey,
			signature.KeyID,
		)
	}

	rawSignature, err := base64.StdEncoding.DecodeString(
		signature.Signature,
	)
	if err != nil {
		return fmt.Errorf(
			"%w: decode signature: %v",
			ErrInvalidPackageSignature,
			err,
		)
	}

	if len(rawSignature) != ed25519.SignatureSize {
		return fmt.Errorf(
			"%w: invalid Ed25519 signature",
			ErrInvalidPackageSignature,
		)
	}

	if !ed25519.Verify(
		publicKey,
		payload,
		rawSignature,
	) {
		return ErrSignatureMismatch
	}

	return nil
}
