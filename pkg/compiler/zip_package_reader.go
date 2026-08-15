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
}

// NewZIPPackageReader creates a ZIP package reader.
func NewZIPPackageReader() *ZIPPackageReader {
	return &ZIPPackageReader{}
}

// NewZIPPackageReaderWithVerifier creates a ZIP package reader
// that verifies package signatures using the supplied verifier.
func NewZIPPackageReaderWithVerifier(
	verifier PackageVerifier,
) *ZIPPackageReader {
	return &ZIPPackageReader{
		verifier: verifier,
	}
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

	bundleData, ok := files[bundleManifestPath]
	if !ok {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: %s is missing",
			ErrInvalidArtifactPackage,
			bundleManifestPath,
		)
	}

	integrityData, ok := files[integrityManifestPath]
	if !ok {
		return ArtifactBundle{}, nil, fmt.Errorf(
			"%w: %s is missing",
			ErrMissingPackageIntegrity,
			integrityManifestPath,
		)
	}

	bundle, err := UnmarshalArtifactBundle(bundleData)
	if err != nil {
		return ArtifactBundle{}, nil, err
	}

	integrity, err := UnmarshalPackageIntegrity(integrityData)
	if err != nil {
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
		if path == bundleManifestPath ||
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

	if err := VerifyPackageIntegrity(
		bundle,
		bundleData,
		payloads,
		integrity,
	); err != nil {
		return ArtifactBundle{}, nil, err
	}

	return bundle, payloads, nil
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
