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
				Module:     "logger",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/logger",
			},
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
			{
				Module:     "web",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
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

func testRunnablePackageBundle() ArtifactBundle {
	return ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Runtime: &RuntimeDescriptor{
			Kind: RuntimeKindApplicationExecutable,
			Entrypoint: RuntimeEntrypoint{
				Module:  "demo",
				Version: "v1",
			},
			TargetOS:   "windows",
			TargetArch: "amd64",
		},
		Artifacts: []Artifact{{
			Module:     "demo",
			Version:    "v1",
			ImportPath: "example.com/demo",
		}},
	}
}

func testRunnablePackagePlaceholderPayloads() map[string][]byte {
	return map[string][]byte{
		"demo@v1": []byte("placeholder-not-executable"),
	}
}

func writeTestPackageV2(t *testing.T, packager *ZIPPackager, path string) {
	t.Helper()
	if err := packager.packageForMetadata(
		testRunnablePackageBundle(),
		testRunnablePackagePlaceholderPayloads(),
		path,
		packageMetadataV2(),
	); err != nil {
		t.Fatal(err)
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

func TestZIPPackagerPublicWriterRemainsPackageFormatV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.zip")
	if err := NewZIPPackager().Package(testPackageBundle(), testPackagePayloads(), path); err != nil {
		t.Fatal(err)
	}
	entries := readZIPEntriesForTest(t, path)
	want := []byte(`{"package_format_version":1,"bundle_schema_version":1}`)
	if !bytes.Equal(entries[packageMetadataPath], want) {
		t.Fatalf("expected %s, got %s", want, entries[packageMetadataPath])
	}
	if _, err := UnmarshalArtifactBundle(entries[bundleManifestPath]); err != nil {
		t.Fatalf("expected canonical v1 bundle: %v", err)
	}
}

func TestZIPPackagerPublicWriterDoesNotAutoSelectV2ForRuntimeBundle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "must-not-write-v2.zip")
	err := NewZIPPackager().Package(
		testRunnablePackageBundle(),
		testRunnablePackagePlaceholderPayloads(),
		path,
	)
	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf("expected ErrInvalidArtifactBundle, got %v", err)
	}
}

func TestZIPPackagerInternalV2WritesCanonicalDocumentsAndIntegrity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v2.zip")
	writeTestPackageV2(t, NewZIPPackager(), path)
	entries := readZIPEntriesForTest(t, path)
	wantPaths := []string{
		"artifacts/demo/v1/artifact",
		"bundle.json",
		"integrity.json",
		"package.json",
	}
	gotPaths := make([]string, 0, len(entries))
	for path := range entries {
		gotPaths = append(gotPaths, path)
	}
	sort.Strings(gotPaths)
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("expected v2 paths %#v, got %#v", wantPaths, gotPaths)
	}

	wantMetadata := []byte(`{"package_format_version":2,"bundle_schema_version":2}`)
	if !bytes.Equal(entries[packageMetadataPath], wantMetadata) {
		t.Fatalf("expected %s, got %s", wantMetadata, entries[packageMetadataPath])
	}
	wantBundle, err := marshalArtifactBundleForSchema(
		testRunnablePackageBundle(), artifactBundleSchemaVersionV2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(entries[bundleManifestPath], wantBundle) {
		t.Fatalf("expected %s, got %s", wantBundle, entries[bundleManifestPath])
	}
	integrity, err := UnmarshalPackageIntegrity(entries[integrityManifestPath])
	if err != nil {
		t.Fatal(err)
	}
	if integrity.Version != packageIntegrityVersion ||
		integrity.PackageMetadataSHA256 != sha256Hex(entries[packageMetadataPath]) ||
		integrity.BundleSHA256 != sha256Hex(entries[bundleManifestPath]) {
		t.Fatalf("unexpected v2 integrity: %#v", integrity)
	}
	if len(integrity.Artifacts) != 1 ||
		integrity.Artifacts[0].SHA256 != sha256Hex([]byte("placeholder-not-executable")) {
		t.Fatalf("unexpected placeholder digest: %#v", integrity.Artifacts)
	}
}

func TestZIPPackagerInternalV2CreatesDeterministicPlaceholderArchive(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first-v2.zip")
	secondPath := filepath.Join(dir, "second-v2.zip")
	packager := NewZIPPackager()
	writeTestPackageV2(t, packager, firstPath)
	writeTestPackageV2(t, packager, secondPath)
	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("v2 placeholder package output is not deterministic")
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
		"package.json",
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

func TestZIPPackagerWritesPackageMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")

	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	); err != nil {
		t.Fatal(err)
	}

	want, err := marshalPackageMetadata(currentPackageMetadata())
	if err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	count := 0
	for _, file := range reader.File {
		if file.Name != packageMetadataPath {
			continue
		}
		count++
		if got := readArchiveFile(t, file); !bytes.Equal(got, want) {
			t.Fatalf("expected package metadata %s, got %s", want, got)
		}
	}

	if count != 1 {
		t.Fatalf("expected package.json exactly once, got %d", count)
	}
}

func TestZIPPackagerBindsPackageMetadataToIntegrity(t *testing.T) {
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

	var metadataJSON, integrityJSON []byte
	for _, file := range reader.File {
		switch file.Name {
		case packageMetadataPath:
			metadataJSON = readArchiveFile(t, file)
		case integrityManifestPath:
			integrityJSON = readArchiveFile(t, file)
		}
	}

	integrity, err := UnmarshalPackageIntegrity(integrityJSON)
	if err != nil {
		t.Fatal(err)
	}
	if integrity.Version != 2 {
		t.Fatalf("expected integrity version 2, got %d", integrity.Version)
	}
	if got, want := integrity.PackageMetadataSHA256, sha256Hex(metadataJSON); got != want {
		t.Fatalf("expected metadata digest %q, got %q", want, got)
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
