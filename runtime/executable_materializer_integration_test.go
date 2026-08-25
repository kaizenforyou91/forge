package runtime

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestSecureExecutableMaterializerMaterializesRealVerifiedExecutable(t *testing.T) {
	repositoryDirectory := repositoryRoot(t)
	fixtureDirectory := filepath.Join(
		repositoryDirectory,
		"pkg",
		"compiler",
		"testdata",
		"runnable_app",
	)
	fixtureSourcePath := filepath.Join(fixtureDirectory, "main.go")
	fixtureSourceBefore, err := os.ReadFile(fixtureSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	fixtureFilesBefore := directoryFiles(t, fixtureDirectory)

	// This fixture composes the public production source registry, OS command
	// runner, Go executable builder, runnable compiler, and signed ZIP packager.
	fixture := buildSignedRunnablePackageFixture(t)
	loader, err := NewVerifiedRunnablePackageLoader(fixture.trustStore)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := loader.Load(fixture.packagePath)
	if err != nil {
		t.Fatal(err)
	}

	entrypoint := compiler.RuntimeEntrypoint{Module: testModule, Version: testVersion}
	if loaded.PackageFormatVersion() != 2 || loaded.BundleSchemaVersion() != 2 {
		t.Fatalf(
			"loaded version pair = (%d,%d), want (2,2)",
			loaded.PackageFormatVersion(),
			loaded.BundleSchemaVersion(),
		)
	}
	if loaded.Entrypoint() != entrypoint {
		t.Fatalf("loaded entrypoint = %#v, want %#v", loaded.Entrypoint(), entrypoint)
	}
	if loaded.ImportPath() != testImportPath {
		t.Fatalf("loaded import path = %q, want %q", loaded.ImportPath(), testImportPath)
	}
	if loaded.TargetOS() != goruntime.GOOS || loaded.TargetArch() != goruntime.GOARCH {
		t.Fatalf("loaded target = %s/%s", loaded.TargetOS(), loaded.TargetArch())
	}
	if loaded.SignerKeyID() != testSignerKeyID {
		t.Fatalf("loaded signer KeyID = %q, want %q", loaded.SignerKeyID(), testSignerKeyID)
	}

	verifiedBytes := loaded.ExecutableBytes()
	if len(verifiedBytes) == 0 {
		t.Fatal("strict loader returned an empty real executable")
	}
	if !bytes.Equal(verifiedBytes, fixture.executable) {
		t.Fatal("strict loader bytes differ from the real packaged executable")
	}
	verifiedSHA256 := sha256.Sum256(verifiedBytes)

	// Replace the source archive after strict loading. Materialization must use
	// only detached VerifiedRunnablePackage state and must not reopen the ZIP.
	if err := os.WriteFile(
		fixture.packagePath,
		[]byte("source package replaced after strict load"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	materializer := NewSecureExecutableMaterializer()
	first, err := materializer.Materialize(loaded)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close() })
	second, err := materializer.Materialize(loaded)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })

	if first.directory == second.directory || first.path == second.path {
		t.Fatalf(
			"real materializations are not isolated: first=%q second=%q",
			first.path,
			second.path,
		)
	}

	assertRealMaterialization(
		t,
		first,
		verifiedBytes,
		verifiedSHA256,
	)
	assertRealMaterialization(
		t,
		second,
		verifiedBytes,
		verifiedSHA256,
	)

	firstDirectory := first.directory
	firstPath := first.path
	secondDirectory := second.directory
	secondPath := second.path
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	for _, removedPath := range []string{
		firstPath,
		firstDirectory,
		secondPath,
		secondDirectory,
	} {
		if _, err := os.Lstat(removedPath); !os.IsNotExist(err) {
			t.Fatalf("materialized path %q still exists or cannot be inspected: %v", removedPath, err)
		}
	}
	if first.Entrypoint() != entrypoint || second.Entrypoint() != entrypoint {
		t.Fatal("entrypoint metadata changed after cleanup")
	}
	if first.SignerKeyID() != testSignerKeyID || second.SignerKeyID() != testSignerKeyID {
		t.Fatal("signer metadata changed after cleanup")
	}
	if first.TargetOS() != goruntime.GOOS || first.TargetArch() != goruntime.GOARCH ||
		second.TargetOS() != goruntime.GOOS || second.TargetArch() != goruntime.GOARCH {
		t.Fatal("target metadata changed after cleanup")
	}
	if err := first.Close(); err != nil {
		t.Fatalf("repeated first Close failed: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("repeated second Close failed: %v", err)
	}

	fixtureSourceAfter, err := os.ReadFile(fixtureSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixtureSourceAfter, fixtureSourceBefore) {
		t.Fatal("real runnable source fixture changed during build or materialization")
	}
	fixtureFilesAfter := directoryFiles(t, fixtureDirectory)
	if !equalStrings(fixtureFilesBefore, fixtureFilesAfter) {
		t.Fatalf(
			"runnable source fixture directory changed: before=%v after=%v",
			fixtureFilesBefore,
			fixtureFilesAfter,
		)
	}
}

func assertRealMaterialization(
	t *testing.T,
	materialized *MaterializedExecutable,
	verifiedBytes []byte,
	verifiedSHA256 [32]byte,
) {
	t.Helper()

	if !filepath.IsAbs(materialized.directory) ||
		!strings.HasPrefix(filepath.Base(materialized.directory), "forge-runtime-") {
		t.Fatalf("unexpected private directory %q", materialized.directory)
	}
	if filepath.Clean(filepath.Dir(materialized.path)) != filepath.Clean(materialized.directory) {
		t.Fatalf(
			"materialized path %q is not directly under %q",
			materialized.path,
			materialized.directory,
		)
	}
	expectedName := "application"
	if goruntime.GOOS == "windows" {
		expectedName = "application.exe"
	}
	if filepath.Base(materialized.path) != expectedName {
		t.Fatalf(
			"materialized filename = %q, want %q",
			filepath.Base(materialized.path),
			expectedName,
		)
	}

	info, err := os.Lstat(materialized.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		t.Fatalf("unexpected materialized file mode %v", info.Mode())
	}
	if info.Size() != int64(len(verifiedBytes)) || info.Size() == 0 {
		t.Fatalf("materialized size = %d, want %d", info.Size(), len(verifiedBytes))
	}
	if goruntime.GOOS != "windows" {
		permissions := info.Mode().Perm()
		if permissions&0o077 != 0 {
			t.Fatalf("materialized file grants group/other permissions %04o", permissions)
		}
		if permissions&0o100 == 0 {
			t.Fatalf("materialized file lacks owner execute permission: %04o", permissions)
		}
	}

	materializedBytes, err := os.ReadFile(materialized.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(materializedBytes, verifiedBytes) {
		t.Fatal("materialized real executable differs from strictly verified bytes")
	}
	if actualSHA256 := sha256.Sum256(materializedBytes); actualSHA256 != verifiedSHA256 {
		t.Fatalf("materialized SHA-256 = %x, want %x", actualSHA256, verifiedSHA256)
	}
}
