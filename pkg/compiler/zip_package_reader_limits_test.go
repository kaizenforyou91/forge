package compiler

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestAlphaRuntimePackageReadLimits(t *testing.T) {
	limits := AlphaRuntimePackageReadLimits()
	if err := limits.Validate(); err != nil {
		t.Fatal(err)
	}
	if limits.MaxArchiveBytes != 80*1024*1024 ||
		limits.MaxEntries != 16 ||
		limits.MaxDocumentBytes != 1*1024*1024 ||
		limits.MaxArtifactBytes != 64*1024*1024 ||
		limits.MaxTotalUncompressedBytes != 72*1024*1024 ||
		!limits.RequireStoreCompression {
		t.Fatalf("unexpected Alpha runtime limits: %#v", limits)
	}
}

func TestPackageReadLimitsValidateRejectsNonPositiveFields(t *testing.T) {
	tests := map[string]func(*PackageReadLimits){
		"archive bytes": func(limits *PackageReadLimits) { limits.MaxArchiveBytes = 0 },
		"entries":       func(limits *PackageReadLimits) { limits.MaxEntries = -1 },
		"document bytes": func(limits *PackageReadLimits) {
			limits.MaxDocumentBytes = 0
		},
		"artifact bytes": func(limits *PackageReadLimits) {
			limits.MaxArtifactBytes = -1
		},
		"total bytes": func(limits *PackageReadLimits) {
			limits.MaxTotalUncompressedBytes = 0
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			limits := testPackageReadLimits()
			mutate(&limits)
			if err := limits.Validate(); !errors.Is(err, ErrInvalidArtifactPackage) {
				t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
			}
		})
	}
}

func TestNewZIPPackageReaderWithPolicyAndVerifierAndLimitsValidatesConfiguration(t *testing.T) {
	limits := testPackageReadLimits()
	limits.MaxEntries = 0
	if _, err := NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
		DefaultPackageVerificationPolicy(), nil, limits,
	); !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}

	if _, err := NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
		StrictPackageVerificationPolicy(), nil, testPackageReadLimits(),
	); !errors.Is(err, ErrPackageVerifierRequired) {
		t.Fatalf("expected ErrPackageVerifierRequired, got %v", err)
	}
}

func TestBoundedZIPPackageReaderRejectsArchiveSizeOverLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.zip")
	if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	limits := AlphaRuntimePackageReadLimits()
	limits.MaxArchiveBytes = info.Size() - 1
	reader := newBoundedPackageReaderForTest(t, limits)
	if _, err := reader.ReadDetailed(path); !errors.Is(err, ErrPackageReadLimitExceeded) {
		t.Fatalf("expected ErrPackageReadLimitExceeded, got %v", err)
	}
}

func TestBoundedZIPPackageReaderRejectsEntryCountOverLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "entries.zip")
	writeZIPEntriesWithMethodForReadLimitTest(t, path, map[string][]byte{
		"one": []byte("1"),
		"two": []byte("2"),
	}, zip.Store)

	limits := testPackageReadLimits()
	limits.MaxEntries = 1
	reader := newBoundedPackageReaderForTest(t, limits)
	if _, err := reader.ReadDetailed(path); !errors.Is(err, ErrPackageReadLimitExceeded) {
		t.Fatalf("expected ErrPackageReadLimitExceeded, got %v", err)
	}
}

func TestBoundedZIPPackageReaderRejectsPackageMetadataOverDocumentLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package-metadata.zip")
	writeZIPEntriesWithMethodForReadLimitTest(t, path, map[string][]byte{
		packageMetadataPath: []byte("{}"),
	}, zip.Store)

	limits := testPackageReadLimits()
	limits.MaxDocumentBytes = 1
	reader := newBoundedPackageReaderForTest(t, limits)
	_, err := reader.ReadDetailed(path)
	if !errors.Is(err, ErrPackageReadLimitExceeded) {
		t.Fatalf("expected ErrPackageReadLimitExceeded, got %v", err)
	}
}

func TestBoundedZIPPackageReaderRejectsBundleOverDocumentLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.zip")
	writeZIPEntriesWithMethodForReadLimitTest(t, path, map[string][]byte{
		bundleManifestPath: []byte("{}"),
	}, zip.Store)

	limits := testPackageReadLimits()
	limits.MaxDocumentBytes = 1
	reader := newBoundedPackageReaderForTest(t, limits)
	_, err := reader.ReadDetailed(path)
	if !errors.Is(err, ErrPackageReadLimitExceeded) {
		t.Fatalf("expected ErrPackageReadLimitExceeded, got %v", err)
	}
}

func TestBoundedZIPPackageReaderRejectsArtifactOverArtifactLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.zip")
	writeZIPEntriesWithMethodForReadLimitTest(t, path, map[string][]byte{
		"artifacts/demo/v1/artifact": []byte("12"),
	}, zip.Store)

	limits := testPackageReadLimits()
	limits.MaxArtifactBytes = 1
	reader := newBoundedPackageReaderForTest(t, limits)
	_, err := reader.ReadDetailed(path)
	if !errors.Is(err, ErrPackageReadLimitExceeded) {
		t.Fatalf("expected ErrPackageReadLimitExceeded, got %v", err)
	}
}

func TestBoundedZIPPackageReaderRejectsTotalUncompressedBytesOverLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "total.zip")
	writeZIPEntriesWithMethodForReadLimitTest(t, path, map[string][]byte{
		"one": []byte("1234"),
		"two": []byte("5678"),
	}, zip.Store)

	limits := testPackageReadLimits()
	limits.MaxTotalUncompressedBytes = 7
	reader := newBoundedPackageReaderForTest(t, limits)
	_, err := reader.ReadDetailed(path)
	if !errors.Is(err, ErrPackageReadLimitExceeded) {
		t.Fatalf("expected ErrPackageReadLimitExceeded, got %v", err)
	}
}

func TestPackageReadLimitRejectsActualZIPEntryBytes(t *testing.T) {
	data, exceeded, err := readAtMost(strings.NewReader("12345"), 4)
	if err != nil {
		t.Fatal(err)
	}
	if !exceeded {
		t.Fatalf("expected actual read limit to be exceeded, got %q", data)
	}
}

func TestBoundedZIPPackageReaderRequiresStoreWithoutChangingGenericReader(t *testing.T) {
	dir := t.TempDir()
	storedPath := filepath.Join(dir, "stored.zip")
	deflatedPath := filepath.Join(dir, "deflated.zip")
	if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), storedPath); err != nil {
		t.Fatal(err)
	}
	writeZIPEntriesForTest(t, deflatedPath, readZIPEntriesForTest(t, storedPath))

	if _, _, err := NewZIPPackageReader().Read(deflatedPath); err != nil {
		t.Fatalf("generic reader must retain Deflate compatibility: %v", err)
	}

	reader := newBoundedPackageReaderForTest(t, AlphaRuntimePackageReadLimits())
	if _, err := reader.ReadDetailed(deflatedPath); !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}
}

func TestBoundedZIPPackageReaderReadsNormalPackageFormats(t *testing.T) {
	dir := t.TempDir()
	v1Path := filepath.Join(dir, "v1.zip")
	v2Path := filepath.Join(dir, "v2.zip")
	if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), v1Path); err != nil {
		t.Fatal(err)
	}
	writeTestPackageV2(t, NewZIPPackager(), v2Path)

	reader := newBoundedPackageReaderForTest(t, AlphaRuntimePackageReadLimits())
	v1, err := reader.ReadDetailed(v1Path)
	if err != nil {
		t.Fatal(err)
	}
	if v1.PackageFormatVersion != packageFormatVersionV1 ||
		v1.BundleSchemaVersion != artifactBundleSchemaVersionV1 {
		t.Fatalf("unexpected v1 evidence: %#v", v1)
	}

	v2, err := reader.ReadDetailed(v2Path)
	if err != nil {
		t.Fatal(err)
	}
	if v2.PackageFormatVersion != packageFormatVersionV2 ||
		v2.BundleSchemaVersion != artifactBundleSchemaVersionV2 {
		t.Fatalf("unexpected v2 evidence: %#v", v2)
	}
}

func TestAlphaRuntimePackageReadLimitsAcceptSignedForgePackages(t *testing.T) {
	dir := t.TempDir()
	signer, verifier := trustedTestSignerAndVerifier(t)
	v1Path := filepath.Join(dir, "signed-v1.zip")
	v2Path := filepath.Join(dir, "signed-v2.zip")
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(), testPackagePayloads(), v1Path,
	); err != nil {
		t.Fatal(err)
	}
	writeTestPackageV2(t, NewZIPPackagerWithSigner(signer), v2Path)

	reader, err := NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
		StrictPackageVerificationPolicy(),
		verifier,
		AlphaRuntimePackageReadLimits(),
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{v1Path, v2Path} {
		result, err := reader.ReadDetailed(path)
		if err != nil {
			t.Fatalf("read %s: %v", filepath.Base(path), err)
		}
		if result.VerifiedSignerKeyID != "forge-dev" {
			t.Fatalf("expected verified signer forge-dev, got %q", result.VerifiedSignerKeyID)
		}
	}
}

func testPackageReadLimits() PackageReadLimits {
	return PackageReadLimits{
		MaxArchiveBytes:           1024 * 1024,
		MaxEntries:                16,
		MaxDocumentBytes:          128 * 1024,
		MaxArtifactBytes:          128 * 1024,
		MaxTotalUncompressedBytes: 512 * 1024,
		RequireStoreCompression:   true,
	}
}

func newBoundedPackageReaderForTest(
	t *testing.T,
	limits PackageReadLimits,
) *ZIPPackageReader {
	t.Helper()

	reader, err := NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
		DefaultPackageVerificationPolicy(),
		nil,
		limits,
	)
	if err != nil {
		t.Fatal(err)
	}

	return reader
}

func writeZIPEntriesWithMethodForReadLimitTest(
	t *testing.T,
	path string,
	entries map[string][]byte,
	method uint16,
) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		entry, err := writer.CreateHeader(&zip.FileHeader{Name: key, Method: method})
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			t.Fatal(err)
		}
		if _, err := entry.Write(entries[key]); err != nil {
			_ = writer.Close()
			_ = file.Close()
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
