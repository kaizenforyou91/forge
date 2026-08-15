package compiler

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// Ed25519Verifier verifies trusted Forge package signatures.
type Ed25519Verifier struct {
	keys map[string]ed25519.PublicKey
}

// NewEd25519Verifier creates an empty trust store.
func NewEd25519Verifier() *Ed25519Verifier {
	return &Ed25519Verifier{
		keys: make(map[string]ed25519.PublicKey),
	}
}

// TrustKey adds a trusted public key.
func (v *Ed25519Verifier) TrustKey(
	keyID string,
	publicKey ed25519.PublicKey,
) error {
	if v == nil {
		return ErrInvalidPackageSignature
	}

	if strings.TrimSpace(keyID) == "" {
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

	if v.keys == nil {
		v.keys = make(map[string]ed25519.PublicKey)
	}

	v.keys[keyID] = append(
		ed25519.PublicKey(nil),
		publicKey...,
	)

	return nil
}

// Verify verifies the signature and ensures the signing key is trusted.
func (v *Ed25519Verifier) Verify(
	payload []byte,
	signature PackageSignature,
) error {
	if v == nil {
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

	trustedKey, ok := v.keys[signature.KeyID]
	if !ok {
		return fmt.Errorf(
			"%w: key %q",
			ErrUntrustedPackageKey,
			signature.KeyID,
		)
	}

	if !ed25519.PublicKey(trustedKey).Equal(
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

	if !ed25519.Verify(
		publicKey,
		payload,
		rawSignature,
	) {
		return ErrSignatureMismatch
	}

	return nil
}
