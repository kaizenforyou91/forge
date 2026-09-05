package compiler

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZIPPackageReader reads Forge deterministic ZIP packages.
type ZIPPackageReader struct {
	verifier PackageVerifier
	policy   PackageVerificationPolicy
	limits   *PackageReadLimits
	lstat    func(string) (os.FileInfo, error)
	open     func(string) (*os.File, error)
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

// NewZIPPackageReaderWithPolicyAndVerifierAndLimits creates a bounded ZIP
// package reader using the supplied verification policy, verifier, and read
// limits. Existing constructors remain unbounded for inspection compatibility.
func NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
	policy PackageVerificationPolicy,
	verifier PackageVerifier,
	limits PackageReadLimits,
) (*ZIPPackageReader, error) {
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	if policy.RequireSignature && verifier == nil {
		return nil, ErrPackageVerifierRequired
	}
	if err := limits.Validate(); err != nil {
		return nil, err
	}

	limitsCopy := limits
	return &ZIPPackageReader{
		verifier: verifier,
		policy:   policy,
		limits:   &limitsCopy,
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
	result, err := r.ReadDetailed(path)
	if err != nil {
		return ArtifactBundle{}, nil, err
	}

	return result.Bundle, result.Payloads, nil
}

// ReadDetailed reads and verifies a ZIP package and returns validated version,
// bundle, payload, and verified-signer evidence.
func (r *ZIPPackageReader) ReadDetailed(
	path string,
) (result PackageReadResult, resultErr error) {
	evidence, err := r.readDetailed(path)
	if err != nil {
		return PackageReadResult{}, err
	}

	return PackageReadResult{
		PackageFormatVersion: evidence.PackageFormatVersion,
		BundleSchemaVersion:  evidence.BundleSchemaVersion,
		Bundle:               evidence.Bundle,
		Payloads:             evidence.Payloads,
		VerifiedSignerKeyID:  evidence.VerifiedSignerKeyID,
	}, nil
}

type packageReadEvidence struct {
	PackageFormatVersion int
	BundleSchemaVersion  int
	Bundle               ArtifactBundle
	Payloads             map[string][]byte
	VerifiedSignerKeyID  string
	IntegrityData        []byte
	Signature            PackageSignature
	SignaturePresent     bool
}

func (r *ZIPPackageReader) readDetailed(
	path string,
) (result packageReadEvidence, resultErr error) {
	if r == nil {
		return packageReadEvidence{}, fmt.Errorf(
			"%w: reader is nil",
			ErrInvalidArtifactPackage,
		)
	}

	if strings.TrimSpace(path) == "" {
		return packageReadEvidence{}, fmt.Errorf(
			"%w: package path is required",
			ErrInvalidArtifactPackage,
		)
	}

	packageFile, reader, err := r.openPackage(path)
	if err != nil {
		return packageReadEvidence{}, err
	}
	defer func() {
		if closeErr := packageFile.Close(); closeErr != nil {
			wrapped := fmt.Errorf(
				"%w: close package: %w",
				ErrInvalidArtifactPackage,
				closeErr,
			)
			if resultErr == nil {
				result = packageReadEvidence{}
				resultErr = wrapped
				return
			}
			resultErr = errors.Join(resultErr, wrapped)
		}
	}()

	if len(reader.File) == 0 {
		return packageReadEvidence{}, fmt.Errorf(
			"%w: package is empty",
			ErrInvalidArtifactPackage,
		)
	}
	if r.limits != nil && len(reader.File) > r.limits.MaxEntries {
		return packageReadEvidence{}, fmt.Errorf(
			"%w: archive contains %d entries, limit is %d",
			ErrPackageReadLimitExceeded,
			len(reader.File),
			r.limits.MaxEntries,
		)
	}

	seen := make(map[string]struct{}, len(reader.File))
	files := make(map[string][]byte, len(reader.File))
	var headerTotal uint64
	var actualTotal int64

	for _, file := range reader.File {
		if !validArchivePath(file.Name) {
			return packageReadEvidence{}, fmt.Errorf(
				"%w: unsafe archive path %q",
				ErrInvalidArtifactPackage,
				file.Name,
			)
		}

		if _, exists := seen[file.Name]; exists {
			return packageReadEvidence{}, fmt.Errorf(
				"%w: duplicate archive entry %q",
				ErrInvalidArtifactPackage,
				file.Name,
			)
		}

		seen[file.Name] = struct{}{}

		if strings.HasSuffix(file.Name, "/") {
			return packageReadEvidence{}, fmt.Errorf(
				"%w: directory entry %q is not allowed",
				ErrInvalidArtifactPackage,
				file.Name,
			)
		}

		var data []byte
		if r.limits == nil {
			data, err = readZIPEntry(file)
		} else {
			if r.limits.RequireStoreCompression && file.Method != zip.Store {
				return packageReadEvidence{}, fmt.Errorf(
					"%w: archive entry %q uses unsupported compression method %d",
					ErrInvalidArtifactPackage,
					file.Name,
					file.Method,
				)
			}

			entryLimit := r.entryReadLimit(file.Name)
			if file.UncompressedSize64 > uint64(entryLimit) {
				return packageReadEvidence{}, fmt.Errorf(
					"%w: archive entry %q declares %d uncompressed bytes, limit is %d",
					ErrPackageReadLimitExceeded,
					file.Name,
					file.UncompressedSize64,
					entryLimit,
				)
			}

			maxTotal := uint64(r.limits.MaxTotalUncompressedBytes)
			if headerTotal > maxTotal || file.UncompressedSize64 > maxTotal-headerTotal {
				return packageReadEvidence{}, fmt.Errorf(
					"%w: declared total uncompressed bytes exceed limit %d at entry %q",
					ErrPackageReadLimitExceeded,
					r.limits.MaxTotalUncompressedBytes,
					file.Name,
				)
			}
			headerTotal += file.UncompressedSize64

			remainingTotal := r.limits.MaxTotalUncompressedBytes - actualTotal
			actualLimit := entryLimit
			totalIsLimiting := remainingTotal < actualLimit
			if totalIsLimiting {
				actualLimit = remainingTotal
			}

			var exceeded bool
			data, exceeded, err = readZIPEntryBounded(file, actualLimit)
			if exceeded {
				if totalIsLimiting {
					return packageReadEvidence{}, fmt.Errorf(
						"%w: actual total uncompressed bytes exceed limit %d at entry %q",
						ErrPackageReadLimitExceeded,
						r.limits.MaxTotalUncompressedBytes,
						file.Name,
					)
				}

				return packageReadEvidence{}, fmt.Errorf(
					"%w: archive entry %q exceeds actual-read limit %d",
					ErrPackageReadLimitExceeded,
					file.Name,
					entryLimit,
				)
			}
			actualTotal += int64(len(data))
		}
		if err != nil {
			return packageReadEvidence{}, fmt.Errorf(
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
		return packageReadEvidence{}, fmt.Errorf(
			"%w: %s is missing",
			ErrLegacyPackageUnsupported,
			packageMetadataPath,
		)
	}

	metadata, err := unmarshalPackageMetadata(packageMetadataData)
	if err != nil {
		return packageReadEvidence{}, err
	}
	bundleSchemaVersion, err := bundleSchemaVersionForPackageMetadata(metadata)
	if err != nil {
		return packageReadEvidence{}, err
	}

	bundleData, ok := files[bundleManifestPath]
	if !ok {
		return packageReadEvidence{}, fmt.Errorf(
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
		return packageReadEvidence{}, err
	}

	var integrityData []byte
	integrityPresent := false

	if data, ok := files[integrityManifestPath]; ok {
		integrityData = data
		integrityPresent = true
	}

	if r.policy.RequireIntegrity && !integrityPresent {
		return packageReadEvidence{}, fmt.Errorf(
			"%w: %s is missing",
			ErrMissingPackageIntegrity,
			integrityManifestPath,
		)
	}

	var integrity PackageIntegrity

	if integrityPresent {
		integrity, err = UnmarshalPackageIntegrity(integrityData)
		if err != nil {
			return packageReadEvidence{}, err
		}
	}
	signature, signaturePresent, err := readOptionalPackageSignature(files)
	if err != nil {
		return packageReadEvidence{}, err
	}

	if signaturePresent && !integrityPresent {
		return packageReadEvidence{}, fmt.Errorf(
			"%w: signature requires %s",
			ErrMissingPackageIntegrity,
			integrityManifestPath,
		)
	}

	verifiedSignerKeyID, err := r.verifySignaturePolicy(
		integrityData,
		signature,
		signaturePresent,
	)
	if err != nil {
		return packageReadEvidence{}, err
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
			return packageReadEvidence{}, fmt.Errorf(
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
			return packageReadEvidence{}, fmt.Errorf(
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
			return packageReadEvidence{}, err
		}
	}

	return packageReadEvidence{
		PackageFormatVersion: metadata.PackageFormatVersion,
		BundleSchemaVersion:  bundleSchemaVersion,
		Bundle:               bundle,
		Payloads:             payloads,
		VerifiedSignerKeyID:  verifiedSignerKeyID,
		IntegrityData:        append([]byte(nil), integrityData...),
		Signature:            signature,
		SignaturePresent:     signaturePresent,
	}, nil
}

func (r *ZIPPackageReader) openPackage(
	path string,
) (*os.File, *zip.Reader, error) {
	lstat := r.lstat
	if lstat == nil {
		lstat = os.Lstat
	}
	open := r.open
	if open == nil {
		open = os.Open
	}

	pathInfo, err := lstat(path)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"%w: inspect package path: %v",
			ErrInvalidArtifactPackage,
			err,
		)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf(
			"%w: package path is a symbolic link",
			ErrInvalidArtifactPackage,
		)
	}
	if !pathInfo.Mode().IsRegular() {
		return nil, nil, fmt.Errorf(
			"%w: package path is not a regular file",
			ErrInvalidArtifactPackage,
		)
	}
	// On Windows, FileInfo obtained from a path may resolve its stable file ID
	// lazily when SameFile is first called. Resolve it while the inspected path
	// still names the selected object, before the package open can be raced.
	if !os.SameFile(pathInfo, pathInfo) {
		return nil, nil, fmt.Errorf(
			"%w: package path identity is unavailable",
			ErrInvalidArtifactPackage,
		)
	}

	packageFile, err := open(path)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"%w: open package: %v",
			ErrInvalidArtifactPackage,
			err,
		)
	}

	openInfo, err := packageFile.Stat()
	if err != nil {
		return nil, nil, closePackageAfterOpenFailure(packageFile, fmt.Errorf(
			"%w: stat open package: %v",
			ErrInvalidArtifactPackage,
			err,
		))
	}
	if !openInfo.Mode().IsRegular() {
		return nil, nil, closePackageAfterOpenFailure(packageFile, fmt.Errorf(
			"%w: open package is not a regular file",
			ErrInvalidArtifactPackage,
		))
	}
	if !os.SameFile(pathInfo, openInfo) {
		return nil, nil, closePackageAfterOpenFailure(packageFile, fmt.Errorf(
			"%w: package path identity changed while opening",
			ErrInvalidArtifactPackage,
		))
	}
	if r.limits != nil && openInfo.Size() > r.limits.MaxArchiveBytes {
		return nil, nil, closePackageAfterOpenFailure(packageFile, fmt.Errorf(
			"%w: archive size %d exceeds limit %d",
			ErrPackageReadLimitExceeded,
			openInfo.Size(),
			r.limits.MaxArchiveBytes,
		))
	}

	reader, err := zip.NewReader(packageFile, openInfo.Size())
	if err != nil {
		return nil, nil, closePackageAfterOpenFailure(packageFile, fmt.Errorf(
			"%w: open package: %v",
			ErrInvalidArtifactPackage,
			err,
		))
	}

	return packageFile, reader, nil
}

func closePackageAfterOpenFailure(packageFile *os.File, primary error) error {
	if closeErr := packageFile.Close(); closeErr != nil {
		return errors.Join(
			primary,
			fmt.Errorf("%w: close package after failure: %w", ErrInvalidArtifactPackage, closeErr),
		)
	}
	return primary
}

func (r *ZIPPackageReader) entryReadLimit(path string) int64 {
	if strings.HasPrefix(path, artifactRootPath+"/") {
		return r.limits.MaxArtifactBytes
	}

	return r.limits.MaxDocumentBytes
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
) (string, error) {
	if !signaturePresent {
		if r.policy.RequireSignature {
			return "", ErrMissingPackageSignature
		}

		return "", nil
	}

	if r.verifier == nil {
		if r.policy.RequireSignature {
			return "", ErrPackageVerifierRequired
		}

		return "", nil
	}

	if err := r.verifier.Verify(payload, signature); err != nil {
		return "", err
	}

	return signature.KeyID, nil
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

func readZIPEntryBounded(
	file *zip.File,
	limit int64,
) ([]byte, bool, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, false, err
	}
	defer reader.Close()

	return readAtMost(reader, limit)
}

func readAtMost(
	reader io.Reader,
	limit int64,
) ([]byte, bool, error) {
	var buffer bytes.Buffer
	chunk := make([]byte, 32*1024)
	var total int64

	for {
		remaining := limit - total
		readSize := len(chunk)
		if remaining < int64(readSize) {
			readSize = int(remaining) + 1
		}

		n, readErr := reader.Read(chunk[:readSize])
		if int64(n) > remaining {
			return nil, true, nil
		}
		if n > 0 {
			_, _ = buffer.Write(chunk[:n])
			total += int64(n)
		}

		if readErr == io.EOF {
			return buffer.Bytes(), false, nil
		}
		if readErr != nil {
			return nil, false, readErr
		}
		if n == 0 {
			return nil, false, io.ErrNoProgress
		}
	}
}
