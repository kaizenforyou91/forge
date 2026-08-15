package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const packageIntegrityVersion = 1

// PackageIntegrity describes cryptographic integrity of a Forge package.
type PackageIntegrity struct {
	Version      int              `json:"version"`
	Algorithm    string           `json:"algorithm"`
	BundleSHA256 string           `json:"bundle_sha256"`
	Artifacts    []ArtifactDigest `json:"artifacts"`
}

// ArtifactDigest contains the SHA-256 digest of one artifact payload.
type ArtifactDigest struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

// BuildPackageIntegrity creates the integrity manifest for a package.
//
// bundleJSON is hashed as stored in bundle.json.
// Artifact payloads are hashed independently by exact module@version identity.
func BuildPackageIntegrity(
	bundle ArtifactBundle,
	bundleJSON []byte,
	payloads map[string][]byte,
) (PackageIntegrity, error) {
	if err := bundle.Validate(); err != nil {
		return PackageIntegrity{}, err
	}

	if bundleJSON == nil {
		return PackageIntegrity{}, fmt.Errorf(
			"%w: bundle.json payload is nil",
			ErrInvalidPackageIntegrity,
		)
	}

	if payloads == nil {
		payloads = map[string][]byte{}
	}

	integrity := PackageIntegrity{
		Version:      packageIntegrityVersion,
		Algorithm:    "sha256",
		BundleSHA256: sha256Hex(bundleJSON),
		Artifacts:    make([]ArtifactDigest, 0, len(bundle.Artifacts)),
	}

	seen := make(map[string]struct{}, len(bundle.Artifacts))

	for _, artifact := range bundle.Artifacts {
		key := artifact.Module + "@" + artifact.Version

		if _, exists := seen[key]; exists {
			return PackageIntegrity{}, fmt.Errorf(
				"%w: duplicate artifact %q",
				ErrInvalidPackageIntegrity,
				key,
			)
		}

		payload, ok := payloads[key]
		if !ok {
			return PackageIntegrity{}, fmt.Errorf(
				"%w: %s",
				ErrMissingArtifactPayload,
				key,
			)
		}

		integrity.Artifacts = append(
			integrity.Artifacts,
			ArtifactDigest{
				Module:  artifact.Module,
				Version: artifact.Version,
				SHA256:  sha256Hex(payload),
			},
		)

		seen[key] = struct{}{}
	}

	if len(payloads) != len(seen) {
		for key := range payloads {
			if _, exists := seen[key]; !exists {
				return PackageIntegrity{}, fmt.Errorf(
					"%w: unexpected artifact payload %q",
					ErrInvalidArtifactPackage,
					key,
				)
			}
		}
	}

	return integrity, nil
}

// Validate validates the integrity manifest contract.
func (p PackageIntegrity) Validate() error {
	if p.Version != packageIntegrityVersion {
		return fmt.Errorf(
			"%w: unsupported integrity version %d",
			ErrInvalidPackageIntegrity,
			p.Version,
		)
	}

	if p.Algorithm != "sha256" {
		return fmt.Errorf(
			"%w: unsupported integrity algorithm %q",
			ErrInvalidPackageIntegrity,
			p.Algorithm,
		)
	}

	if !validSHA256(p.BundleSHA256) {
		return fmt.Errorf(
			"%w: invalid bundle SHA-256 digest",
			ErrInvalidPackageIntegrity,
		)
	}

	seen := make(map[string]struct{}, len(p.Artifacts))

	for i, artifact := range p.Artifacts {
		if artifact.Module == "" || artifact.Version == "" {
			return fmt.Errorf(
				"%w: artifact %d has invalid identity",
				ErrInvalidPackageIntegrity,
				i,
			)
		}

		if !validSHA256(artifact.SHA256) {
			return fmt.Errorf(
				"%w: artifact %q has invalid SHA-256 digest",
				ErrInvalidPackageIntegrity,
				artifact.Module+"@"+artifact.Version,
			)
		}

		key := artifact.Module + "@" + artifact.Version

		if _, exists := seen[key]; exists {
			return fmt.Errorf(
				"%w: duplicate artifact %q",
				ErrInvalidPackageIntegrity,
				key,
			)
		}

		seen[key] = struct{}{}
	}

	return nil
}

// MarshalPackageIntegrity serializes the integrity manifest deterministically.
func MarshalPackageIntegrity(
	integrity PackageIntegrity,
) ([]byte, error) {
	if err := integrity.Validate(); err != nil {
		return nil, err
	}

	data, err := json.Marshal(integrity)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: marshal integrity manifest: %v",
			ErrInvalidPackageIntegrity,
			err,
		)
	}

	return data, nil
}

// UnmarshalPackageIntegrity deserializes and validates an integrity manifest.
func UnmarshalPackageIntegrity(
	data []byte,
) (PackageIntegrity, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return PackageIntegrity{}, fmt.Errorf(
			"%w: empty integrity document",
			ErrInvalidPackageIntegrity,
		)
	}

	var integrity PackageIntegrity

	if err := json.Unmarshal(data, &integrity); err != nil {
		return PackageIntegrity{}, fmt.Errorf(
			"%w: invalid integrity JSON: %v",
			ErrInvalidPackageIntegrity,
			err,
		)
	}

	if err := integrity.Validate(); err != nil {
		return PackageIntegrity{}, err
	}

	return integrity, nil
}

// VerifyPackageIntegrity verifies bundle and payload digests.
func VerifyPackageIntegrity(
	bundle ArtifactBundle,
	bundleJSON []byte,
	payloads map[string][]byte,
	integrity PackageIntegrity,
) error {
	if err := integrity.Validate(); err != nil {
		return err
	}

	if err := bundle.Validate(); err != nil {
		return err
	}

	if actual := sha256Hex(bundleJSON); actual != integrity.BundleSHA256 {
		return fmt.Errorf(
			"%w: bundle.json SHA-256 mismatch",
			ErrIntegrityMismatch,
		)
	}

	if len(integrity.Artifacts) != len(bundle.Artifacts) {
		return fmt.Errorf(
			"%w: integrity artifact count %d does not match bundle artifact count %d",
			ErrIntegrityMismatch,
			len(integrity.Artifacts),
			len(bundle.Artifacts),
		)
	}

	for i, artifact := range bundle.Artifacts {
		expected := integrity.Artifacts[i]

		key := artifact.Module + "@" + artifact.Version

		if expected.Module != artifact.Module ||
			expected.Version != artifact.Version {
			return fmt.Errorf(
				"%w: integrity artifact %d does not match bundle artifact %q",
				ErrIntegrityMismatch,
				i,
				key,
			)
		}

		payload, ok := payloads[key]
		if !ok {
			return fmt.Errorf(
				"%w: %s",
				ErrMissingArtifactPayload,
				key,
			)
		}

		actual := sha256Hex(payload)

		if actual != expected.SHA256 {
			return fmt.Errorf(
				"%w: artifact %q SHA-256 mismatch",
				ErrIntegrityMismatch,
				key,
			)
		}
	}

	if len(payloads) != len(bundle.Artifacts) {
		return fmt.Errorf(
			"%w: payload count mismatch",
			ErrIntegrityMismatch,
		)
	}

	return nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}

	_, err := hex.DecodeString(value)
	return err == nil
}
