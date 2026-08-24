package compiler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ZIP artifact layout:
//
//	package.json
//	bundle.json
//	integrity.json
//	artifacts/<module>/<version>/artifact
//
// All entries use a fixed timestamp so repeated packaging of the same input
// produces byte-identical archives.
const (
	packageMetadataPath   = "package.json"
	bundleManifestPath    = "bundle.json"
	integrityManifestPath = "integrity.json"
	signatureManifestPath = "signature.json"
	artifactRootPath      = "artifacts"
	artifactFileName      = "artifact"
)

// ZIPPackager creates deterministic artifact archives.
type ZIPPackager struct {
	signer PackageSigner
}

// NewZIPPackager creates a ZIP packager.
func NewZIPPackager() *ZIPPackager {
	return &ZIPPackager{}
}

func NewZIPPackagerWithSigner(
	signer PackageSigner,
) *ZIPPackager {
	return &ZIPPackager{
		signer: signer,
	}
}

// Package writes an ArtifactBundle, its integrity manifest, and payloads
// into a deterministic ZIP.
//
// Payloads are keyed by exact artifact identity:
//
//	module@version
//
// Every bundle artifact must have exactly one payload.
// Extra payloads are rejected to prevent silent packaging inconsistencies.
func (p *ZIPPackager) Package(
	bundle ArtifactBundle,
	payloads map[string][]byte,
	outputPath string,
) error {
	return p.packageForMetadata(
		bundle,
		payloads,
		outputPath,
		packageMetadataV1(),
	)
}

func (p *ZIPPackager) packageForMetadata(
	bundle ArtifactBundle,
	payloads map[string][]byte,
	outputPath string,
	metadata packageMetadataDocument,
) error {
	if p == nil {
		return fmt.Errorf(
			"%w: packager is nil",
			ErrInvalidArtifactPackage,
		)
	}

	if err := validateSupportedPackageMetadata(metadata); err != nil {
		return err
	}

	if err := bundle.ValidateForSchema(metadata.BundleSchemaVersion); err != nil {
		return err
	}

	if strings.TrimSpace(outputPath) == "" {
		return fmt.Errorf(
			"%w: output path is required",
			ErrInvalidArtifactPackage,
		)
	}

	if payloads == nil {
		payloads = map[string][]byte{}
	}

	expected := make(map[string]struct{}, len(bundle.Artifacts))

	for _, artifact := range bundle.Artifacts {
		key := artifact.Module + "@" + artifact.Version

		expected[key] = struct{}{}

		if _, ok := payloads[key]; !ok {
			return fmt.Errorf(
				"%w: %s",
				ErrMissingArtifactPayload,
				key,
			)
		}
	}

	if len(payloads) != len(expected) {
		for key := range payloads {
			if _, ok := expected[key]; !ok {
				return fmt.Errorf(
					"%w: unexpected artifact payload %q",
					ErrInvalidArtifactPackage,
					key,
				)
			}
		}
	}

	packageMetadataJSON, err := marshalPackageMetadata(metadata)
	if err != nil {
		return err
	}

	bundleJSON, err := marshalArtifactBundleForSchema(
		bundle,
		metadata.BundleSchemaVersion,
	)
	if err != nil {
		return err
	}

	integrity, err := buildPackageIntegrityForSchema(
		metadata.BundleSchemaVersion,
		packageMetadataJSON,
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		return err
	}

	integrityJSON, err := MarshalPackageIntegrity(integrity)
	if err != nil {
		return err
	}

	entries := make([]zipPackageEntry, 0, len(bundle.Artifacts)+4)

	entries = append(entries,
		zipPackageEntry{
			path:    packageMetadataPath,
			payload: append([]byte(nil), packageMetadataJSON...),
		},
		zipPackageEntry{
			path:    bundleManifestPath,
			payload: append([]byte(nil), bundleJSON...),
		},
		zipPackageEntry{
			path:    integrityManifestPath,
			payload: append([]byte(nil), integrityJSON...),
		},
	)

	if p.signer != nil {
		signature, err := p.signer.Sign(integrityJSON)
		if err != nil {
			return fmt.Errorf(
				"%w: sign package integrity: %v",
				ErrInvalidPackageSignature,
				err,
			)
		}

		signatureJSON, err := MarshalPackageSignature(signature)
		if err != nil {
			return err
		}

		entries = append(entries, zipPackageEntry{
			path:    signatureManifestPath,
			payload: signatureJSON,
		})
	}

	for _, artifact := range bundle.Artifacts {
		key := artifact.Module + "@" + artifact.Version

		entries = append(entries, zipPackageEntry{
			path: filepath.ToSlash(filepath.Join(
				artifactRootPath,
				artifact.Module,
				artifact.Version,
				artifactFileName,
			)),
			payload: append([]byte(nil), payloads[key]...),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})

	data, err := buildDeterministicZIP(entries)
	if err != nil {
		return fmt.Errorf(
			"%w: build ZIP: %v",
			ErrInvalidArtifactPackage,
			err,
		)
	}

	if err := writePackageAtomic(outputPath, data); err != nil {
		return fmt.Errorf(
			"%w: write %q: %v",
			ErrInvalidArtifactPackage,
			outputPath,
			err,
		)
	}

	return nil
}

type zipPackageEntry struct {
	path    string
	payload []byte
}

func buildDeterministicZIP(entries []zipPackageEntry) ([]byte, error) {
	var buffer bytes.Buffer

	writer := zip.NewWriter(&buffer)

	fixedTime := time.Unix(0, 0).UTC()

	for _, entry := range entries {
		if !validArchivePath(entry.path) {
			_ = writer.Close()

			return nil, fmt.Errorf(
				"unsafe archive path %q",
				entry.path,
			)
		}

		header := &zip.FileHeader{
			Name:     entry.path,
			Method:   zip.Store,
			Modified: fixedTime,
		}

		zipEntryWriter, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}

		if _, err := zipEntryWriter.Write(entry.payload); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func validArchivePath(path string) bool {
	if path == "" {
		return false
	}

	path = filepath.ToSlash(path)

	if strings.HasPrefix(path, "/") {
		return false
	}

	if strings.Contains(path, "../") ||
		strings.HasPrefix(path, "../") ||
		path == ".." {
		return false
	}

	parts := strings.Split(path, "/")

	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}

	return true
}

func writePackageAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}

	file, err := os.CreateTemp(directory, ".forge-package-*")
	if err != nil {
		return err
	}

	tempPath := file.Name()

	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(tempPath)
	}

	if _, err := file.Write(data); err != nil {
		cleanup()
		return err
	}

	if err := file.Sync(); err != nil {
		cleanup()
		return err
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if err := os.Chmod(tempPath, 0o644); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	return nil
}
