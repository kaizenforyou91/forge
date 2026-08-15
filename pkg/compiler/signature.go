package compiler

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// PackageSigner signs package integrity metadata.
//
// The signed payload is the exact serialized integrity.json bytes.
// Signing integrity.json transitively protects bundle.json and all
// artifact payloads because their SHA-256 digests are contained in it.
type PackageSigner interface {
	Sign(payload []byte) (PackageSignature, error)
}

// PackageVerifier verifies a package signature.
type PackageVerifier interface {
	Verify(payload []byte, signature PackageSignature) error
}

// PackageSignature describes the cryptographic signature of a Forge package.
type PackageSignature struct {
	Version   int    `json:"version"`
	Algorithm string `json:"algorithm"`
	KeyID     string `json:"key_id"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

const packageSignatureVersion = 1
const packageSignatureAlgorithm = "ed25519"

// Validate validates the package signature structure.
func (s PackageSignature) Validate() error {
	if s.Version != packageSignatureVersion {
		return fmt.Errorf(
			"%w: unsupported signature version %d",
			ErrInvalidPackageSignature,
			s.Version,
		)
	}

	if s.Algorithm != packageSignatureAlgorithm {
		return fmt.Errorf(
			"%w: unsupported signature algorithm %q",
			ErrInvalidPackageSignature,
			s.Algorithm,
		)
	}

	if strings.TrimSpace(s.KeyID) == "" {
		return fmt.Errorf(
			"%w: key ID is required",
			ErrInvalidPackageSignature,
		)
	}

	publicKey, err := base64.StdEncoding.DecodeString(s.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf(
			"%w: invalid Ed25519 public key",
			ErrInvalidPackageSignature,
		)
	}

	signature, err := base64.StdEncoding.DecodeString(s.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return fmt.Errorf(
			"%w: invalid Ed25519 signature",
			ErrInvalidPackageSignature,
		)
	}

	return nil
}
