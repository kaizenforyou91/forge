package compiler

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

func decodeEd25519PublicKey(
	value string,
) (ed25519.PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: decode public key: %v",
			ErrInvalidPackageSignature,
			err,
		)
	}

	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"%w: invalid Ed25519 public key",
			ErrInvalidPackageSignature,
		)
	}

	return ed25519.PublicKey(
		append([]byte(nil), data...),
	), nil
}