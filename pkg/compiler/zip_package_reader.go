package compiler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// ZIPPackageReader reads Forge deterministic ZIP packages.
type ZIPPackageReader struct {
	verifier PackageVerifier
	policy   PackageVerificationPolicy
}

// NewZIPPackageReader creates a ZIP package reader.
func NewZIPPackageReader() *ZIPPackageReader {
	return &ZIPPackageReader{
		policy: DefaultPackageVerificationPolicy(),
	}
}

// NewZIPPackageReaderWithVerifier creates a ZIP package reader
// that verifies package signatures using the supplied verifier.
func NewZIPPackageReaderWithVerifier(
	verifier PackageVerifier,
) *ZIPPackageReader {
	return &ZIPPackageReader{
		verifier: verifier,
		policy:   DefaultPackageVerificationPolicy(),
	}
}

// NewZIPPackageReaderWithPolicy creates a ZIP package reader
// using the supplied verification policy.
func NewZIPPackageReaderWithPolicy(
	policy PackageVerificationPolicy,
) (*ZIPPackageReader, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}

	return &ZIPPackageReader{
		policy: policy,
	}, nil
}

// NewZIPPackageReaderWithPolicyAndVerifier creates a ZIP package reader
// using both verification policy and package verifier.
func NewZIPPackageReaderWithPolicyAndVerifier(
	policy PackageVerificationPolicy,
	verifier PackageVerifier,
) (*ZIPPackageReader, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}

	if policy.RequireSignature && verifier == nil {
		return nil, ErrPackageVerifierRequired
	}

	return &ZIPPackageReader{
		verifier: verifier,
		policy:   policy,
	}, nil
}

// Read reads and verifies a ZIP package.
//
// It returns:
//   - the validated artifact bundle
//   - artifact payloads keyed by module@version
//
// The input archive is never modified.
func (r *ZIPPackageReader) Read(
	path string,
) (ArtifactBundle, map[string][]byte, error) {
	if r == nil {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: reader is nil",
			ErrInvalidArtifactPackage,
		)
	}

	if strings.TrimSpace(path) == "" {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: package path is required",
			ErrInvalidArtifactPackage,
		)
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: open package: %v",
			ErrInvalidArtifactPackage,
			err,
		)
	}
	defer reader.Close()

	if len(reader.File) == 0 {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: package is empty",
			ErrInvalidArtifactPackage,
		)
	}

	seen := make(map[string]struct{}, len(reader.File))
	files := make(map[string][]byte, len(reader.File))

	for _, file := range reader.File {
		if !validArchivePath(file.Name) {
			return ArtifactBundle{}, nil, fmt.Errorf(
				"%w: unsafe archive path %q",
				ErrInvalidArtifactPackage,
				file.Name,
			)
		}

		if _, exists := seen[file.Name]; exists {
			return ArtifactBundle{}, nil, fmt.Errorf(
				"%w: duplicate archive entry %q",
				ErrInvalidArtifactPackage,
				file.Name,
			)
		}

		seen[file.Name] = struct{}{}

		if strings.HasSuffix(file.Name, "/") {
			return ArtifactBundle{}, nil, fmt.Errorf(
				"%w: directory entry %q is not allowed",
				ErrInvalidArtifactPackage,
				file.Name,
			)
		}

		data, err := readZIPEntry(file)
		if err != nil {
			return ArtifactBundle{}, nil, fmt.Errorf(
				"%w: read %q: %v",
				ErrInvalidArtifactPackage,
				file.Name,
				err,
			)
		}

		files[file.Name] = data
	}

	packageMetadataData, ok := files[packageMetadataPath]
	if !ok {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: %s is missing",
			ErrLegacyPackageUnsupported,
			packageMetadataPath,
		)
	}

	metadata, err := unmarshalPackageMetadata(packageMetadataData)
	if err != nil {
		return ArtifactBundle{}, nil, err
	}
	bundleSchemaVersion, err := bundleSchemaVersionForPackageMetadata(metadata)
	if err != nil {
		return ArtifactBundle{}, nil, err
	}

	bundleData, ok := files[bundleManifestPath]
	if !ok {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: %s is missing",
			ErrInvalidArtifactPackage,
			bundleManifestPath,
		)
	}

	bundle, err := unmarshalArtifactBundleForSchema(
		bundleData,
		bundleSchemaVersion,
	)
	if err != nil {
		return ArtifactBundle{}, nil, err
	}

	var integrityData []byte
	integrityPresent := false

	if data, ok := files[integrityManifestPath]; ok {
		integrityData = data
		integrityPresent = true
	}

	if r.policy.RequireIntegrity && !integrityPresent {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: %s is missing",
			ErrMissingPackageIntegrity,
			integrityManifestPath,
		)
	}

	var integrity PackageIntegrity

	if integrityPresent {
		integrity, err = UnmarshalPackageIntegrity(integrityData)
		if err != nil {
			return ArtifactBundle{}, nil, err
		}
	}
	signature, signaturePresent, err := readOptionalPackageSignature(files)
	if err != nil {
		return ArtifactBundle{}, nil, err
	}

	if signaturePresent && !integrityPresent {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: signature requires %s",
			ErrMissingPackageIntegrity,
			integrityManifestPath,
		)
	}

	if err := r.verifySignaturePolicy(
		integrityData,
		signature,
		signaturePresent,
	); err != nil {
		return ArtifactBundle{}, nil, err
	}

	expected := make(map[string]string, len(bundle.Artifacts))
	payloads := make(map[string][]byte, len(bundle.Artifacts))

	for _, artifact := range bundle.Artifacts {
		key := artifact.Module + "@" + artifact.Version

		archivePath := filepath.ToSlash(filepath.Join(
			artifactRootPath,
			artifact.Module,
			artifact.Version,
			artifactFileName,
		))

		expected[archivePath] = key

		payload, ok := files[archivePath]
		if !ok {
			return ArtifactBundle{}, nil, fmt.Errorf(
				"%w: artifact payload %q is missing",
				ErrMissingArtifactPayload,
				key,
			)
		}

		payloads[key] = append([]byte(nil), payload...)
	}

	for path := range files {
		if path == packageMetadataPath ||
			path == bundleManifestPath ||
			path == integrityManifestPath ||
			path == signatureManifestPath {
			continue
		}

		if _, ok := expected[path]; !ok {
			return ArtifactBundle{}, nil, fmt.Errorf(
				"%w: unexpected package entry %q",
				ErrInvalidArtifactPackage,
				path,
			)
		}
	}
	if r.policy.RequireIntegrity {
		if err := verifyPackageIntegrityForSchema(
			bundleSchemaVersion,
			packageMetadataData,
			bundle,
			bundleData,
			payloads,
			integrity,
		); err != nil {
			return ArtifactBundle{}, nil, err
		}
	}

	return bundle, payloads, nil
}

func bundleSchemaVersionForPackageMetadata(metadata packageMetadataDocument) (int, error) {
	switch metadata {
	case packageMetadataV1():
		return artifactBundleSchemaVersionV1, nil
	case packageMetadataV2():
		return artifactBundleSchemaVersionV2, nil
	default:
		return 0, fmt.Errorf(
			"%w: package format version %d / bundle schema version %d has no archive dispatch",
			ErrUnsupportedPackageFormat,
			metadata.PackageFormatVersion,
			metadata.BundleSchemaVersion,
		)
	}
}

func (r *ZIPPackageReader) verifySignaturePolicy(
	payload []byte,
	signature PackageSignature,
	signaturePresent bool,
) error {
	if !signaturePresent {
		if r.policy.RequireSignature {
			return ErrMissingPackageSignature
		}

		return nil
	}

	if r.verifier == nil {
		if r.policy.RequireSignature {
			return ErrPackageVerifierRequired
		}

		return nil
	}

	if err := r.verifier.Verify(payload, signature); err != nil {
		return err
	}

	return nil
}

func readOptionalPackageSignature(
	files map[string][]byte,
) (PackageSignature, bool, error) {
	signatureData, ok := files[signatureManifestPath]
	if !ok {
		return PackageSignature{}, false, nil
	}

	signature, err := UnmarshalPackageSignature(signatureData)
	if err != nil {
		return PackageSignature{}, true, err
	}

	return signature, true, nil
}

func readZIPEntry(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var buffer bytes.Buffer

	if _, err := io.Copy(&buffer, reader); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
