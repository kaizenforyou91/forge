package compiler

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

func TestNewZIPPackageReader(t *testing.T) {
	reader := NewZIPPackageReader()

	if reader == nil {
		t.Fatal("expected non-nil ZIP package reader")
	}
}

func TestZIPPackageReaderRejectsSymlinkAndNonRegularPackagePaths(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target.zip")
		if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), target); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(directory, "link.zip")
		if err := os.Symlink(target, link); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("Windows symlink creation is unavailable: %v", err)
			}
			t.Fatal(err)
		}
		if _, err := NewZIPPackageReader().ReadDetailed(link); !errors.Is(err, ErrInvalidArtifactPackage) {
			t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
		}
	})

	t.Run("directory", func(t *testing.T) {
		if _, err := NewZIPPackageReader().ReadDetailed(t.TempDir()); !errors.Is(err, ErrInvalidArtifactPackage) {
			t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
		}
	})

	t.Run("socket", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix-domain filesystem sockets are unavailable on Windows")
		}
		path := filepath.Join(t.TempDir(), "package.zip")
		listener, err := net.Listen("unix", path)
		if err != nil {
			t.Skipf("Unix-domain socket creation is unavailable: %v", err)
		}
		defer listener.Close()
		if _, err := NewZIPPackageReader().ReadDetailed(path); !errors.Is(err, ErrInvalidArtifactPackage) {
			t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
		}
	})
}

func TestZIPPackageReaderRejectsReplacementBetweenLstatAndOpen(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "selected.zip")
	replacement := filepath.Join(directory, "replacement.zip")
	displaced := filepath.Join(directory, "displaced.zip")
	if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), path); err != nil {
		t.Fatal(err)
	}
	if err := NewZIPPackager().Package(ArtifactBundle{
		ManifestName: "replacement", ManifestVersion: "v1",
	}, nil, replacement); err != nil {
		t.Fatal(err)
	}

	reader := NewZIPPackageReader()
	reader.open = func(requested string) (*os.File, error) {
		if err := os.Rename(requested, displaced); err != nil {
			return nil, err
		}
		if err := os.Rename(replacement, requested); err != nil {
			return nil, err
		}
		return os.Open(requested)
	}

	if _, err := reader.ReadDetailed(path); !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf("expected identity-bound ErrInvalidArtifactPackage, got %v", err)
	}
}

func TestZIPPackageReaderRejectsPackageTruncatedAfterOpen(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "selected.zip")
	if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), path); err != nil {
		t.Fatal(err)
	}

	reader := NewZIPPackageReader()
	reader.open = func(requested string) (*os.File, error) {
		file, err := os.Open(requested)
		if err != nil {
			return nil, err
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			return nil, err
		}
		if err := os.Truncate(requested, info.Size()/2); err != nil {
			_ = file.Close()
			return nil, err
		}
		return file, nil
	}

	if _, err := reader.ReadDetailed(path); !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf("expected truncated opened package to fail safely, got %v", err)
	}
}

func TestZIPPackageReaderPostOpenReplacementCannotRedirectSameKeyRead(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ordinary Windows read handles prevent deterministic rename replacement in this fixture")
	}

	directory := t.TempDir()
	path := filepath.Join(directory, "selected.zip")
	replacement := filepath.Join(directory, "replacement.zip")
	displaced := filepath.Join(directory, "displaced.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)
	writeSignedIdentityPackageForOpenTest(t, signer, path, "selected")
	writeSignedIdentityPackageForOpenTest(t, signer, replacement, "replacement")

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(), verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	reader.open = func(requested string) (*os.File, error) {
		file, err := os.Open(requested)
		if err != nil {
			return nil, err
		}
		if err := os.Rename(requested, displaced); err != nil {
			_ = file.Close()
			return nil, err
		}
		if err := os.Rename(replacement, requested); err != nil {
			_ = file.Close()
			return nil, err
		}
		return file, nil
	}

	result, err := reader.ReadDetailed(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Bundle.ManifestName != "selected" {
		t.Fatalf("read pathname replacement %q instead of opened object", result.Bundle.ManifestName)
	}
}

func writeSignedIdentityPackageForOpenTest(
	t *testing.T,
	signer *Ed25519Signer,
	path string,
	name string,
) {
	t.Helper()
	bundle := ArtifactBundle{ManifestName: name, ManifestVersion: "v1"}
	if err := NewZIPPackagerWithSigner(signer).Package(bundle, nil, path); err != nil {
		t.Fatal(err)
	}
}

func TestZIPPackageReaderStillReadsPackageFormatV1(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	if err := NewZIPPackager().Package(
		bundle,
		payloads,
		path,
	); err != nil {
		t.Fatal(err)
	}

	gotBundle, gotPayloads, err := NewZIPPackageReader().Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotBundle, bundle) {
		t.Fatalf(
			"expected bundle %#v, got %#v",
			bundle,
			gotBundle,
		)
	}

	if !reflect.DeepEqual(gotPayloads, payloads) {
		t.Fatalf(
			"expected payloads %#v, got %#v",
			payloads,
			gotPayloads,
		)
	}
	if gotBundle.Runtime != nil {
		t.Fatalf("expected v1 runtime metadata to remain nil, got %#v", gotBundle.Runtime)
	}
}

func TestZIPPackageReaderReadDetailedReturnsPackageFormatV1Evidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.zip")
	if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), path); err != nil {
		t.Fatal(err)
	}

	result, err := NewZIPPackageReader().ReadDetailed(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.PackageFormatVersion != packageFormatVersionV1 {
		t.Fatalf("expected package format version 1, got %d", result.PackageFormatVersion)
	}
	if result.BundleSchemaVersion != artifactBundleSchemaVersionV1 {
		t.Fatalf("expected bundle schema version 1, got %d", result.BundleSchemaVersion)
	}
	if !reflect.DeepEqual(result.Bundle, testPackageBundle()) {
		t.Fatalf("unexpected v1 bundle: %#v", result.Bundle)
	}
	if !reflect.DeepEqual(result.Payloads, testPackagePayloads()) {
		t.Fatalf("unexpected v1 payloads: %#v", result.Payloads)
	}
}

func TestZIPPackageReaderReadsPackageFormatV2PlaceholderPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2.zip")
	writeTestPackageV2(t, NewZIPPackager(), path)

	gotBundle, gotPayloads, err := NewZIPPackageReader().Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotBundle, testRunnablePackageBundle()) {
		t.Fatalf("expected bundle %#v, got %#v", testRunnablePackageBundle(), gotBundle)
	}
	if !reflect.DeepEqual(gotPayloads, testRunnablePackagePlaceholderPayloads()) {
		t.Fatalf("expected placeholder payloads %#v, got %#v", testRunnablePackagePlaceholderPayloads(), gotPayloads)
	}
}

func TestZIPPackageReaderReadDetailedReturnsPackageFormatV2Evidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2.zip")
	writeTestPackageV2(t, NewZIPPackager(), path)

	result, err := NewZIPPackageReader().ReadDetailed(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.PackageFormatVersion != packageFormatVersionV2 {
		t.Fatalf("expected package format version 2, got %d", result.PackageFormatVersion)
	}
	if result.BundleSchemaVersion != artifactBundleSchemaVersionV2 {
		t.Fatalf("expected bundle schema version 2, got %d", result.BundleSchemaVersion)
	}
	if !reflect.DeepEqual(result.Bundle, testRunnablePackageBundle()) {
		t.Fatalf("unexpected v2 bundle: %#v", result.Bundle)
	}
	if !reflect.DeepEqual(result.Payloads, testRunnablePackagePlaceholderPayloads()) {
		t.Fatalf("unexpected v2 payloads: %#v", result.Payloads)
	}
}

func TestZIPPackageReaderReadDetailedRejectsUnsupportedVersionPairs(t *testing.T) {
	for name, metadata := range map[string][]byte{
		"cross pair 1 2": []byte(`{"package_format_version":1,"bundle_schema_version":2}`),
		"cross pair 2 1": []byte(`{"package_format_version":2,"bundle_schema_version":1}`),
		"future package": []byte(`{"package_format_version":3,"bundle_schema_version":2}`),
		"future bundle":  []byte(`{"package_format_version":2,"bundle_schema_version":3}`),
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "unsupported.zip")
			writeZIPEntriesForTest(t, path, map[string][]byte{
				packageMetadataPath: metadata,
			})

			_, err := NewZIPPackageReader().ReadDetailed(path)
			if !errors.Is(err, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
			}
		})
	}
}

func TestZIPPackageReaderRejectsInvalidBundleV2(t *testing.T) {
	validBundle, err := marshalArtifactBundleForSchema(
		testRunnablePackageBundle(), artifactBundleSchemaVersionV2,
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string][]byte{
		"missing runtime":      []byte(`{"manifest_name":"demo","manifest_version":"v1","artifacts":[{"module":"demo","version":"v1","import_path":"example.com/demo"}]}`),
		"unknown runtime kind": bytes.Replace(validBundle, []byte(`"application_executable"`), []byte(`"unknown"`), 1),
		"missing target os":    bytes.Replace(validBundle, []byte(`"target_os":"windows",`), nil, 1),
		"missing target arch":  bytes.Replace(validBundle, []byte(`,"target_arch":"amd64"`), nil, 1),
		"unmatched entrypoint": bytes.Replace(validBundle, []byte(`"module":"demo"`), []byte(`"module":"missing"`), 1),
		"unknown field":        bytes.Replace(validBundle, []byte(`"artifacts":`), []byte(`"future":true,"artifacts":`), 1),
		"duplicate nested key": bytes.Replace(validBundle, []byte(`"target_os":"windows"`), []byte(`"target_os":"windows","target_os":"linux"`), 1),
	}

	metadata, err := marshalPackageMetadata(packageMetadataV2())
	if err != nil {
		t.Fatal(err)
	}
	for name, bundleJSON := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid-v2.zip")
			writeZIPEntriesForTest(t, path, map[string][]byte{
				packageMetadataPath: metadata,
				bundleManifestPath:  bundleJSON,
			})
			_, _, err := NewZIPPackageReader().Read(path)
			if !errors.Is(err, ErrInvalidArtifactBundle) {
				t.Fatalf("expected ErrInvalidArtifactBundle, got %v", err)
			}
		})
	}
}

func TestZIPPackageReaderRequiresBundleJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.zip")

	if err := createZIP(
		path,
		map[string][]byte{
			packageMetadataPath:          mustPackageMetadataJSON(t),
			"artifacts/http/v1/artifact": []byte("http"),
		},
	); err != nil {
		t.Fatal(err)
	}

	_, _, err := NewZIPPackageReader().Read(path)
	if err == nil {
		t.Fatal("expected missing bundle.json error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRequiresPackageMetadata(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	legacyPath := filepath.Join(dir, "legacy.zip")
	createValidFW051Package(t, validPath)

	entries := readZIPEntriesForTest(t, validPath)
	delete(entries, packageMetadataPath)
	writeZIPEntriesForTest(t, legacyPath, entries)

	_, _, err := NewZIPPackageReader().Read(legacyPath)
	if !errors.Is(err, ErrLegacyPackageUnsupported) {
		t.Fatalf("expected ErrLegacyPackageUnsupported, got %v", err)
	}
}

func TestZIPPackageReaderRejectsUnsupportedPackageFormat(t *testing.T) {
	requireZIPPackageReaderMetadataError(
		t,
		[]byte(`{"package_format_version":2,"bundle_schema_version":1}`),
		ErrUnsupportedPackageFormat,
	)
}

func TestZIPPackageReaderRejectsUnsupportedBundleSchema(t *testing.T) {
	requireZIPPackageReaderMetadataError(
		t,
		[]byte(`{"package_format_version":1,"bundle_schema_version":2}`),
		ErrUnsupportedPackageFormat,
	)
}

func TestZIPPackageReaderRejectsInvalidPackageMetadata(t *testing.T) {
	for _, metadata := range [][]byte{
		[]byte(`{"package_format_version":`),
		[]byte(`{"package_format_version":1,"bundle_schema_version":1,"unknown":true}`),
		[]byte(`{"package_format_version":1,"package_format_version":1,"bundle_schema_version":1}`),
	} {
		requireZIPPackageReaderMetadataError(t, metadata, ErrInvalidPackageMetadata)
	}
}

func TestZIPPackageReaderDetectsPackageMetadataIntegrityMismatch(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	tamperedPath := filepath.Join(dir, "tampered.zip")
	createValidFW051Package(t, validPath)

	entries := readZIPEntriesForTest(t, validPath)
	entries[packageMetadataPath] = append(entries[packageMetadataPath], ' ')
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err := NewZIPPackageReader().Read(tamperedPath)
	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
	}
}

func TestZIPPackageReaderV2TamperMatrix(t *testing.T) {
	for name, mutate := range map[string]func(map[string][]byte){
		"package metadata exact bytes": func(entries map[string][]byte) {
			entries[packageMetadataPath] = append(entries[packageMetadataPath], ' ')
		},
		"runtime metadata": func(entries map[string][]byte) {
			entries[bundleManifestPath] = bytes.Replace(
				entries[bundleManifestPath],
				[]byte(`"target_arch":"amd64"`),
				[]byte(`"target_arch":"arm64"`),
				1,
			)
		},
		"placeholder payload": func(entries map[string][]byte) {
			entries["artifacts/demo/v1/artifact"] = []byte("changed-placeholder")
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			validPath := filepath.Join(dir, "valid-v2.zip")
			tamperedPath := filepath.Join(dir, "tampered-v2.zip")
			writeTestPackageV2(t, NewZIPPackager(), validPath)
			entries := readZIPEntriesForTest(t, validPath)
			mutate(entries)
			writeZIPEntriesForTest(t, tamperedPath, entries)
			_, _, err := NewZIPPackageReader().Read(tamperedPath)
			if !errors.Is(err, ErrIntegrityMismatch) {
				t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
			}
		})
	}
}

func TestZIPPackageReaderRejectsPackageVersionStrippingAndDowngrade(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
		want     error
	}{
		{
			name:     "missing package format version",
			metadata: `{"bundle_schema_version":1}`,
			want:     ErrInvalidPackageMetadata,
		},
		{
			name:     "missing bundle schema version",
			metadata: `{"package_format_version":1}`,
			want:     ErrInvalidPackageMetadata,
		},
		{
			name:     "zero package format version",
			metadata: `{"package_format_version":0,"bundle_schema_version":1}`,
			want:     ErrUnsupportedPackageFormat,
		},
		{
			name:     "negative package format version",
			metadata: `{"package_format_version":-1,"bundle_schema_version":1}`,
			want:     ErrUnsupportedPackageFormat,
		},
		{
			name:     "future package format version",
			metadata: `{"package_format_version":3,"bundle_schema_version":1}`,
			want:     ErrUnsupportedPackageFormat,
		},
		{
			name:     "unsupported bundle schema version",
			metadata: `{"package_format_version":1,"bundle_schema_version":3}`,
			want:     ErrUnsupportedPackageFormat,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireZIPPackageReaderMetadataError(
				t,
				[]byte(test.metadata),
				test.want,
			)
		})
	}
}

func TestZIPPackageReaderRejectsIntegrityVersionDowngradeAndFuture(t *testing.T) {
	for _, version := range []int{0, 1, packageIntegrityVersion + 1} {
		t.Run(fmt.Sprintf("version_%d", version), func(t *testing.T) {
			dir := t.TempDir()
			validPath := filepath.Join(dir, "valid.zip")
			changedPath := filepath.Join(dir, "changed.zip")
			createValidFW051Package(t, validPath)

			entries := readZIPEntriesForTest(t, validPath)
			entries[integrityManifestPath] = bytes.Replace(
				entries[integrityManifestPath],
				[]byte(`"version":2`),
				[]byte(fmt.Sprintf(`"version":%d`, version)),
				1,
			)
			writeZIPEntriesForTest(t, changedPath, entries)

			_, _, err := NewZIPPackageReader().Read(changedPath)
			if !errors.Is(err, ErrInvalidPackageIntegrity) {
				t.Fatalf("expected ErrInvalidPackageIntegrity, got %v", err)
			}
		})
	}
}

func TestZIPPackageReaderRejectsDuplicateIntegrityDigestBeforeComparison(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	changedPath := filepath.Join(dir, "duplicate-integrity-digest.zip")
	createValidFW051Package(t, validPath)

	entries := readZIPEntriesForTest(t, validPath)
	entries[integrityManifestPath] = bytes.Replace(
		entries[integrityManifestPath],
		[]byte(`"bundle_sha256":`),
		[]byte(`"bundle_sha256":null,"bundle_sha256":`),
		1,
	)
	writeZIPEntriesForTest(t, changedPath, entries)

	_, _, err := NewZIPPackageReader().Read(changedPath)
	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf("expected ErrInvalidPackageIntegrity, got %v", err)
	}
	if errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("duplicate digest reached integrity comparison: %v", err)
	}
}

func TestZIPPackageReaderTamperMatrix(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string][]byte)
	}{
		{
			name: "bundle json",
			mutate: func(entries map[string][]byte) {
				entries[bundleManifestPath] = append(entries[bundleManifestPath], ' ')
			},
		},
		{
			name: "artifact payload",
			mutate: func(entries map[string][]byte) {
				entries["artifacts/http/v1/artifact"] = []byte("tampered")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			validPath := filepath.Join(dir, "valid.zip")
			tamperedPath := filepath.Join(dir, "tampered.zip")
			createValidFW051Package(t, validPath)

			entries := readZIPEntriesForTest(t, validPath)
			test.mutate(entries)
			writeZIPEntriesForTest(t, tamperedPath, entries)

			_, _, err := NewZIPPackageReader().Read(tamperedPath)
			if !errors.Is(err, ErrIntegrityMismatch) {
				t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
			}
		})
	}
}

func TestZIPPackageReaderStableErrorPrecedence(t *testing.T) {
	t.Run("unsafe structure before metadata presence", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "unsafe.zip")
		writeZIPEntriesForTest(t, path, map[string][]byte{
			"../escape": []byte("unsafe"),
		})
		_, _, err := NewZIPPackageReader().Read(path)
		if !errors.Is(err, ErrInvalidArtifactPackage) {
			t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
		}
	})

	t.Run("metadata presence before bundle presence", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing-documents.zip")
		writeZIPEntriesForTest(t, path, map[string][]byte{
			"placeholder": []byte("value"),
		})
		_, _, err := NewZIPPackageReader().Read(path)
		if !errors.Is(err, ErrLegacyPackageUnsupported) {
			t.Fatalf("expected ErrLegacyPackageUnsupported, got %v", err)
		}
	})

	t.Run("metadata decode before bundle presence", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid-metadata.zip")
		writeZIPEntriesForTest(t, path, map[string][]byte{
			packageMetadataPath: []byte(`{"package_format_version":`),
		})
		_, _, err := NewZIPPackageReader().Read(path)
		if !errors.Is(err, ErrInvalidPackageMetadata) {
			t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
		}
	})

	t.Run("bundle decode before integrity policy", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid-bundle.zip")
		writeZIPEntriesForTest(t, path, map[string][]byte{
			packageMetadataPath: mustPackageMetadataJSON(t),
			bundleManifestPath:  []byte(`{"manifest_name":""}`),
		})
		_, _, err := NewZIPPackageReader().Read(path)
		if !errors.Is(err, ErrInvalidArtifactBundle) {
			t.Fatalf("expected ErrInvalidArtifactBundle, got %v", err)
		}
	})
}

func TestZIPPackageReaderRejectsMissingPayload(t *testing.T) {
	dir := t.TempDir()

	validPath := filepath.Join(dir, "valid.zip")
	brokenPath := filepath.Join(dir, "broken.zip")

	createValidFW051Package(t, validPath)

	reader, err := zip.OpenReader(validPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	entries := make(map[string][]byte)

	for _, file := range reader.File {
		data, err := readZIPEntry(file)
		if err != nil {
			t.Fatal(err)
		}

		if file.Name == "artifacts/http/v1/artifact" {
			continue
		}

		entries[file.Name] = data
	}

	writeZIPEntriesForTest(t, brokenPath, entries)

	packageReader := NewZIPPackageReader()

	_, _, err = packageReader.Read(brokenPath)
	if err == nil {
		t.Fatal("expected missing artifact payload error")
	}

	if !errors.Is(err, ErrMissingArtifactPayload) {
		t.Fatalf(
			"expected ErrMissingArtifactPayload, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRejectsUnexpectedPayload(t *testing.T) {
	dir := t.TempDir()

	validPath := filepath.Join(dir, "valid.zip")
	brokenPath := filepath.Join(dir, "broken.zip")

	createValidFW051Package(t, validPath)

	reader, err := zip.OpenReader(validPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	entries := make(map[string][]byte)

	for _, file := range reader.File {
		data, err := readZIPEntry(file)
		if err != nil {
			t.Fatal(err)
		}

		entries[file.Name] = data
	}

	entries["artifacts/unknown/v1/artifact"] = []byte("unexpected")

	writeZIPEntriesForTest(t, brokenPath, entries)

	packageReader := NewZIPPackageReader()

	_, _, err = packageReader.Read(brokenPath)
	if err == nil {
		t.Fatal("expected unexpected artifact package entry error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRejectsDuplicateEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "duplicate.zip")

	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
		},
	}

	bundleData, err := MarshalArtifactBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}

	if err := createZIPWithDuplicate(
		path,
		"bundle.json",
		bundleData,
	); err != nil {
		t.Fatal(err)
	}

	_, _, err = NewZIPPackageReader().Read(path)
	if err == nil {
		t.Fatal("expected duplicate entry error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRejectsTraversalEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "traversal.zip")

	if err := createZIP(
		path,
		map[string][]byte{
			"../evil": []byte("bad"),
		},
	); err != nil {
		t.Fatal(err)
	}

	_, _, err := NewZIPPackageReader().Read(path)
	if err == nil {
		t.Fatal("expected traversal error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRejectsInvalidBundleJSON(t *testing.T) {
	dir := t.TempDir()

	validPath := filepath.Join(dir, "valid.zip")
	brokenPath := filepath.Join(dir, "broken.zip")

	createValidFW051Package(t, validPath)

	reader, err := zip.OpenReader(validPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	entries := make(map[string][]byte)

	for _, file := range reader.File {
		data, err := readZIPEntry(file)
		if err != nil {
			t.Fatal(err)
		}

		if file.Name == bundleManifestPath {
			data = []byte(`{"manifest_name":`)
		}

		entries[file.Name] = data
	}

	writeZIPEntriesForTest(t, brokenPath, entries)

	packageReader := NewZIPPackageReader()

	_, _, err = packageReader.Read(brokenPath)
	if err == nil {
		t.Fatal("expected invalid bundle JSON error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRejectsCorruptZIP(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.zip")

	if err := os.WriteFile(
		path,
		[]byte("not a zip archive"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	_, _, err := NewZIPPackageReader().Read(path)
	if err == nil {
		t.Fatal("expected corrupt ZIP error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRejectsEmptyZIP(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.zip")

	if err := createZIP(path, nil); err != nil {
		t.Fatal(err)
	}

	_, _, err := NewZIPPackageReader().Read(path)
	if err == nil {
		t.Fatal("expected empty ZIP error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRejectsNilReceiver(t *testing.T) {
	var reader *ZIPPackageReader

	_, _, err := reader.Read("bundle.zip")
	if err == nil {
		t.Fatal("expected nil receiver error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderRejectsEmptyPath(t *testing.T) {
	err := func() error {
		_, _, err := NewZIPPackageReader().Read("")
		return err
	}()

	if err == nil {
		t.Fatal("expected empty path error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderPreservesPayloadCopies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	if err := NewZIPPackager().Package(
		bundle,
		payloads,
		path,
	); err != nil {
		t.Fatal(err)
	}

	_, gotPayloads, err := NewZIPPackageReader().Read(path)
	if err != nil {
		t.Fatal(err)
	}

	gotPayloads["http@v1"][0] = 'X'

	if bytes.Equal(
		gotPayloads["http@v1"],
		payloads["http@v1"],
	) {
		t.Fatal("reader returned aliased payload data")
	}
}

func TestZIPPackageReaderReadDetailedOwnsPayloadCopies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bundle.zip")
	if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), path); err != nil {
		t.Fatal(err)
	}

	first, err := NewZIPPackageReader().ReadDetailed(path)
	if err != nil {
		t.Fatal(err)
	}
	first.Payloads["http@v1"][0] = 'X'
	delete(first.Payloads, "logger@v1")

	second, err := NewZIPPackageReader().ReadDetailed(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(second.Payloads, testPackagePayloads()) {
		t.Fatalf("detailed read payloads aliased prior result: %#v", second.Payloads)
	}
}

func createZIP(
	path string,
	entries map[string][]byte,
) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	writer := zip.NewWriter(file)

	for name, payload := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			return err
		}

		if _, err := entry.Write(payload); err != nil {
			_ = writer.Close()
			_ = file.Close()
			return err
		}
	}

	if err := writer.Close(); err != nil {
		_ = file.Close()
		return err
	}

	return file.Close()
}

func mustPackageMetadataJSON(t *testing.T) []byte {
	t.Helper()

	data, err := marshalPackageMetadata(currentPackageMetadata())
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func readZIPEntriesForTest(t *testing.T, path string) map[string][]byte {
	t.Helper()

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	entries := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		data, err := readZIPEntry(file)
		if err != nil {
			t.Fatal(err)
		}
		entries[file.Name] = data
	}

	return entries
}

func requireZIPPackageReaderMetadataError(
	t *testing.T,
	metadataJSON []byte,
	want error,
) {
	t.Helper()

	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	changedPath := filepath.Join(dir, "changed.zip")
	createValidFW051Package(t, validPath)

	entries := readZIPEntriesForTest(t, validPath)
	entries[packageMetadataPath] = metadataJSON
	writeZIPEntriesForTest(t, changedPath, entries)

	_, _, err := NewZIPPackageReader().Read(changedPath)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func createZIPWithDuplicate(
	path string,
	name string,
	payload []byte,
) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	writer := zip.NewWriter(file)

	for i := 0; i < 2; i++ {
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			return err
		}

		if _, err := entry.Write(payload); err != nil {
			_ = writer.Close()
			_ = file.Close()
			return err
		}
	}

	if err := writer.Close(); err != nil {
		_ = file.Close()
		return err
	}

	return file.Close()
}

func writeZIPEntriesForTest(
	t *testing.T,
	path string,
	entries map[string][]byte,
) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)

	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		entry, err := writer.Create(key)
		if err != nil {
			writer.Close()
			t.Fatal(err)
		}

		if _, err := entry.Write(entries[key]); err != nil {
			writer.Close()
			t.Fatal(err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

func createValidFW051Package(
	t *testing.T,
	path string,
) {
	t.Helper()

	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:     "http",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
				Version:    "v1",
			},
			{
				Module:     "logger",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/logger",
				Version:    "v1",
			},
			{
				Module:     "web",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
				Version:    "v1",
			},
		},
	}

	payloads := map[string][]byte{
		"http@v1":   []byte("http-artifact"),
		"logger@v1": []byte("logger-artifact"),
		"web@v1":    []byte("web-artifact"),
	}

	packager := NewZIPPackager()

	if err := packager.Package(
		bundle,
		payloads,
		path,
	); err != nil {
		t.Fatal(err)
	}
}
