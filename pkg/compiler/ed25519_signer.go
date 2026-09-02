package compiler

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Ed25519Signer signs Forge package integrity metadata.
type Ed25519Signer struct {
	keyID   string
	private ed25519.PrivateKey
	public  ed25519.PublicKey
}

// NewEd25519Signer creates a signer from an Ed25519 private key.
func NewEd25519Signer(
	keyID string,
	privateKey ed25519.PrivateKey,
) (*Ed25519Signer, error) {
	if err := ValidateKeyID(keyID); err != nil {
		return nil, fmt.Errorf(
			"%w: invalid signing key ID: %w",
			ErrInvalidPackageSignature,
			err,
		)
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf(
			"%w: invalid Ed25519 private key length",
			ErrInvalidPackageSignature,
		)
	}

	privateCopy := append(
		ed25519.PrivateKey(nil),
		privateKey...,
	)

	publicKey := privateCopy.Public().(ed25519.PublicKey)

	return &Ed25519Signer{
		keyID:   keyID,
		private: privateCopy,
		public:  append(ed25519.PublicKey(nil), publicKey...),
	}, nil
}

// GenerateEd25519Signer generates a new random Ed25519 key pair.
func GenerateEd25519Signer(
	keyID string,
) (*Ed25519Signer, error) {
	if err := ValidateKeyID(keyID); err != nil {
		return nil, fmt.Errorf(
			"%w: invalid signing key ID: %w",
			ErrInvalidPackageSignature,
			err,
		)
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: generate Ed25519 key pair: %v",
			ErrInvalidPackageSignature,
			err,
		)
	}

	signer, err := NewEd25519Signer(keyID, privateKey)
	if err != nil {
		return nil, err
	}

	signer.public = append(
		ed25519.PublicKey(nil),
		publicKey...,
	)

	return signer, nil
}

// Sign creates a detached Ed25519 package signature.
func (s *Ed25519Signer) Sign(payload []byte) (PackageSignature, error) {
	if s == nil || len(s.private) != ed25519.PrivateKeySize {
		return PackageSignature{}, ErrInvalidPackageSignature
	}

	signature := ed25519.Sign(s.private, payload)

	result := PackageSignature{
		Version:   packageSignatureVersion,
		Algorithm: packageSignatureAlgorithm,
		KeyID:     s.keyID,
		PublicKey: base64.StdEncoding.EncodeToString(s.public),
		Signature: base64.StdEncoding.EncodeToString(signature),
	}

	if err := result.Validate(); err != nil {
		return PackageSignature{}, err
	}

	return result, nil
}
