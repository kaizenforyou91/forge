package compiler

import (
	"archive/zip"
	"bytes"
	"errors"
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

func TestZIPPackageReaderReadsPackage(t *testing.T) {
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
