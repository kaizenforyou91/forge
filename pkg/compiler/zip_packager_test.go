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

func testPackageBundle() ArtifactBundle {
	return ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:  "logger",
				Version: "v1",
			},
			{
				Module:  "http",
				Version: "v1",
			},
			{
				Module:  "web",
				Version: "v1",
			},
		},
	}
}

func testPackagePayloads() map[string][]byte {
	return map[string][]byte{
		"logger@v1": []byte("logger-payload"),
		"http@v1":   []byte("http-payload"),
		"web@v1":    []byte("web-payload"),
	}
}

func TestNewZIPPackager(t *testing.T) {
	packager := NewZIPPackager()

	if packager == nil {
		t.Fatal("expected non-nil ZIP packager")
	}
}

func TestZIPPackagerCreatesDeterministicArchive(t *testing.T) {
	dir := t.TempDir()

	firstPath := filepath.Join(dir, "first.zip")
	secondPath := filepath.Join(dir, "second.zip")

	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	packager := NewZIPPackager()

	if err := packager.Package(bundle, payloads, firstPath); err != nil {
		t.Fatal(err)
	}

	if err := packager.Package(bundle, payloads, secondPath); err != nil {
		t.Fatal(err)
	}

	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}

	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("ZIP packaging is not deterministic")
	}
}

func TestZIPPackagerContainsExpectedEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	got := make([]string, 0, len(reader.File))

	for _, file := range reader.File {
		got = append(got, file.Name)
	}

	want := []string{
		"artifacts/http/v1/artifact",
		"artifacts/logger/v1/artifact",
		"artifacts/web/v1/artifact",
		"bundle.json",
		"integrity.json",
	}

	sort.Strings(got)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"expected archive entries %#v, got %#v",
			want,
			got,
		)
	}
}

func TestZIPPackagerPreservesPayloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	expected := testPackagePayloads()

	for _, file := range reader.File {
		if file.Name == "bundle.json" {
			continue
		}

		key := filepath.ToSlash(filepath.Clean(file.Name))

		switch key {
		case "artifacts/http/v1/artifact":
			checkArchivePayload(t, file, expected["http@v1"])
		case "artifacts/logger/v1/artifact":
			checkArchivePayload(t, file, expected["logger@v1"])
		case "artifacts/web/v1/artifact":
			checkArchivePayload(t, file, expected["web@v1"])
		}
	}
}

func TestZIPPackagerIncludesSerializedBundle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != "bundle.json" {
			continue
		}

		data := readArchiveFile(t, file)

		decoded, err := UnmarshalArtifactBundle(data)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(decoded, testPackageBundle()) {
			t.Fatalf(
				"decoded bundle does not match expected bundle: %#v",
				decoded,
			)
		}

		return
	}

	t.Fatal("bundle.json not found")
}

func TestZIPPackagerRejectsMissingPayload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	payloads := testPackagePayloads()
	delete(payloads, "http@v1")

	err := NewZIPPackager().Package(
		testPackageBundle(),
		payloads,
		path,
	)
	if err == nil {
		t.Fatal("expected missing payload error")
	}

	if !errors.Is(err, ErrMissingArtifactPayload) {
		t.Fatalf(
			"expected ErrMissingArtifactPayload, got %v",
			err,
		)
	}
}

func TestZIPPackagerRejectsUnexpectedPayload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	payloads := testPackagePayloads()
	payloads["other@v1"] = []byte("unexpected")

	err := NewZIPPackager().Package(
		testPackageBundle(),
		payloads,
		path,
	)
	if err == nil {
		t.Fatal("expected unexpected payload error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestZIPPackagerRejectsInvalidBundle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	bundle := ArtifactBundle{
		ManifestName:    "",
		ManifestVersion: "v1",
	}

	err := NewZIPPackager().Package(
		bundle,
		nil,
		path,
	)
	if err == nil {
		t.Fatal("expected invalid bundle error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestZIPPackagerAllowsEmptyBundle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.zip")

	bundle := ArtifactBundle{
		ManifestName:    "empty",
		ManifestVersion: "v1",
	}

	if err := NewZIPPackager().Package(
		bundle,
		nil,
		path,
	); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Size() == 0 {
		t.Fatal("expected non-empty ZIP archive")
	}
}

func TestZIPPackagerCreatesParentDirectories(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(
		dir,
		"dist",
		"nested",
		"bundle.zip",
	)

	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestZIPPackagerOverwritesDeterministically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	packager := NewZIPPackager()
	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	if err := packager.Package(bundle, payloads, path); err != nil {
		t.Fatal(err)
	}

	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	payloads["web@v1"] = []byte("changed")

	if err := packager.Package(bundle, payloads, path); err != nil {
		t.Fatal(err)
	}

	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(first, second) {
		t.Fatal("expected changed payload to change archive")
	}
}

func TestZIPPackagerDoesNotMutateInputs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	originalBundle := bundle
	originalPayloads := clonePayloads(payloads)

	if err := NewZIPPackager().Package(
		bundle,
		payloads,
		path,
	); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(bundle, originalBundle) {
		t.Fatal("packager mutated bundle")
	}

	if !reflect.DeepEqual(payloads, originalPayloads) {
		t.Fatal("packager mutated payload map")
	}
}

func TestZIPPackagerRejectsNilReceiver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	var packager *ZIPPackager

	err := packager.Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	)
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

func TestValidArchivePathRejectsTraversal(t *testing.T) {
	invalid := []string{
		"",
		".",
		"..",
		"../artifact",
		"artifacts/../artifact",
		"/absolute/path",
		"artifacts//artifact",
	}

	for _, path := range invalid {
		if validArchivePath(path) {
			t.Fatalf("expected invalid archive path: %q", path)
		}
	}
}

func TestValidArchivePathAcceptsNormalPath(t *testing.T) {
	valid := []string{
		"bundle.json",
		"artifacts/http/v1/artifact",
		"artifacts/web/v2/artifact",
	}

	for _, path := range valid {
		if !validArchivePath(path) {
			t.Fatalf("expected valid archive path: %q", path)
		}
	}
}

func checkArchivePayload(
	t *testing.T,
	file *zip.File,
	want []byte,
) {
	t.Helper()

	got := readArchiveFile(t, file)

	if !bytes.Equal(got, want) {
		t.Fatalf(
			"expected payload %q, got %q",
			want,
			got,
		)
	}
}

func clonePayloads(
	input map[string][]byte,
) map[string][]byte {
	result := make(map[string][]byte, len(input))

	for key, payload := range input {
		result[key] = append([]byte(nil), payload...)
	}

	return result
}

func readArchiveFile(
	t *testing.T,
	file *zip.File,
) []byte {
	t.Helper()

	reader, err := file.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	buffer := new(bytes.Buffer)

	if _, err := buffer.ReadFrom(reader); err != nil {
		t.Fatal(err)
	}

	return buffer.Bytes()
}
