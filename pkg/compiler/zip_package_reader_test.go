package compiler

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestNewZIPPackageReader(t *testing.T) {
	reader := NewZIPPackageReader()

	if reader == nil {
		t.Fatal("expected non-nil ZIP package reader")
	}
}

func TestZIPPackageReaderReadsCurrentPackageFormat(t *testing.T) {
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
			metadata: `{"package_format_version":2,"bundle_schema_version":1}`,
			want:     ErrUnsupportedPackageFormat,
		},
		{
			name:     "unsupported bundle schema version",
			metadata: `{"package_format_version":1,"bundle_schema_version":2}`,
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
