package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MarshalPackageSignature serializes a package signature deterministically.
func MarshalPackageSignature(
	signature PackageSignature,
) ([]byte, error) {
	if err := signature.Validate(); err != nil {
		return nil, err
	}

	data, err := json.Marshal(signature)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: marshal package signature: %v",
			ErrInvalidPackageSignature,
			err,
		)
	}

	return data, nil
}

// UnmarshalPackageSignature deserializes and validates a package signature.
func UnmarshalPackageSignature(
	data []byte,
) (PackageSignature, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return PackageSignature{}, fmt.Errorf(
			"%w: empty signature document",
			ErrInvalidPackageSignature,
		)
	}

	var signature PackageSignature

	if err := decodeStrictJSON(data, &signature); err != nil {
		return PackageSignature{}, fmt.Errorf(
			"%w: invalid signature JSON: %v",
			ErrInvalidPackageSignature,
			err,
		)
	}

	if err := signature.Validate(); err != nil {
		return PackageSignature{}, err
	}

	return signature, nil
}
