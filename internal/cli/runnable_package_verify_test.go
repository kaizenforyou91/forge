package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/kaizenforyou91/forge/pkg/manifest"
)

const runnableVerificationTestImportPath = "github.com/kaizenforyou91/forge/pkg/compiler/testdata/runnable_app"

type runnableVerificationFixture struct {
	path     string
	expected runnablePackageExpectation
}

func TestRunnablePackageVerificationAcceptsStrictSignedPackageV2(t *testing.T) {
	fixture := buildRunnableVerificationFixture(t, true)
	before, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyStagedRunnablePackage(fixture.path, fixture.expected); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	afterInfo, err := os.Stat(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || !os.SameFile(beforeInfo, afterInfo) {
		t.Fatal("verification modified or replaced the staged package")
	}
}

func TestRunnablePackageVerificationRejectsSemanticMismatches(t *testing.T) {
	fixture := buildRunnableVerificationFixture(t, true)
	_, otherPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]func(*runnablePackageExpectation){
		"wrong key ID": func(expectation *runnablePackageExpectation) {
			expectation.KeyID = "other-signer"
		},
		"wrong public key": func(expectation *runnablePackageExpectation) {
			expectation.PublicKey = append(
				ed25519.PublicKey(nil),
				otherPrivateKey.Public().(ed25519.PublicKey)...,
			)
		},
		"wrong manifest name": func(expectation *runnablePackageExpectation) {
			expectation.ManifestName = "other-manifest"
		},
		"wrong manifest version": func(expectation *runnablePackageExpectation) {
			expectation.ManifestVersion = "v9"
		},
		"wrong entrypoint": func(expectation *runnablePackageExpectation) {
			expectation.Entrypoint = compiler.RuntimeEntrypoint{Module: "other", Version: "v1"}
		},
		"wrong target OS": func(expectation *runnablePackageExpectation) {
			expectation.TargetOS = "other-os"
		},
		"wrong target architecture": func(expectation *runnablePackageExpectation) {
			expectation.TargetArch = "other-arch"
		},
		"wrong import path": func(expectation *runnablePackageExpectation) {
			expectation.ImportPath = "example.com/other/app"
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			expected := fixture.expected
			expected.PublicKey = append(ed25519.PublicKey(nil), fixture.expected.PublicKey...)
			mutate(&expected)
			err := verifyStagedRunnablePackage(fixture.path, expected)
			requireRunnableVerificationError(t, err)
		})
	}
}

func TestRunnablePackageVerificationPreservesCryptographicErrors(t *testing.T) {
	signed := buildRunnableVerificationFixture(t, true)
	wrongKey := signed.expected
	_, otherPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	wrongKey.PublicKey = otherPrivateKey.Public().(ed25519.PublicKey)
	err = verifyStagedRunnablePackage(signed.path, wrongKey)
	requireRunnableVerificationError(t, err)
	if !errors.Is(err, compiler.ErrUntrustedPackageKey) {
		t.Fatalf("expected ErrUntrustedPackageKey, got %v", err)
	}

	unsigned := buildRunnableVerificationFixture(t, false)
	err = verifyStagedRunnablePackage(unsigned.path, unsigned.expected)
	requireRunnableVerificationError(t, err)
	if !errors.Is(err, compiler.ErrMissingPackageSignature) {
		t.Fatalf("expected ErrMissingPackageSignature, got %v", err)
	}
}

func TestRunnablePackageVerificationRejectsTamperedPackage(t *testing.T) {
	fixture := buildRunnableVerificationFixture(t, true)
	data, err := os.ReadFile(fixture.path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("signed package is empty")
	}
	data[len(data)/2] ^= 0xff
	tamperedPath := filepath.Join(t.TempDir(), "tampered.zip")
	if err := os.WriteFile(tamperedPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	requireRunnableVerificationError(
		t,
		verifyStagedRunnablePackage(tamperedPath, fixture.expected),
	)
}

func TestRunnablePackageVerificationRejectsSignedPackageV1(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const keyID = "runnable-verification-v1"
	signer, err := compiler.NewEd25519Signer(keyID, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "signed-v1.zip")
	bundle := compiler.ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []compiler.Artifact{
			{Module: "app", Version: "v1", ImportPath: "example.com/app"},
		},
	}
	if err := compiler.NewZIPPackagerWithSigner(signer).Package(
		bundle,
		map[string][]byte{"app@v1": []byte("identity")},
		path,
	); err != nil {
		t.Fatal(err)
	}

	err = verifyStagedRunnablePackage(path, runnablePackageExpectation{
		KeyID:           keyID,
		PublicKey:       privateKey.Public().(ed25519.PublicKey),
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Entrypoint:      compiler.RuntimeEntrypoint{Module: "app", Version: "v1"},
		TargetOS:        runtime.GOOS,
		TargetArch:      runtime.GOARCH,
		ImportPath:      "example.com/app",
	})
	requireRunnableVerificationError(t, err)
}

func TestRunnablePackageVerificationRejectsIncompleteExpectation(t *testing.T) {
	fixture := buildRunnableVerificationFixture(t, true)
	for name, expected := range map[string]runnablePackageExpectation{
		"empty":               {},
		"missing public key":  fixture.expected,
		"missing import path": fixture.expected,
	} {
		t.Run(name, func(t *testing.T) {
			switch name {
			case "missing public key":
				expected.PublicKey = nil
			case "missing import path":
				expected.ImportPath = ""
			}
			requireRunnableVerificationError(
				t,
				verifyStagedRunnablePackage(fixture.path, expected),
			)
		})
	}
}

func buildRunnableVerificationFixture(
	t *testing.T,
	signed bool,
) runnableVerificationFixture {
	t.Helper()
	t.Setenv("GOTELEMETRY", "off")

	repositoryRoot := runnableVerificationRepositoryRoot(t)
	entrypoint := manifest.ApplicationEntrypoint{Module: "runnable-app", Version: "v1"}
	m := manifest.Manifest{
		Version:    "v1",
		Name:       "runnable-verification",
		Entrypoint: &entrypoint,
		Modules: []manifest.Module{
			{
				Name:       entrypoint.Module,
				Version:    entrypoint.Version,
				ImportPath: runnableVerificationTestImportPath,
			},
		},
	}
	admission, err := compiler.PrepareManifestAdmission(m, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyID := "runnable-verification-test"
	signer, err := compiler.NewEd25519Signer(keyID, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	packager := compiler.NewZIPPackager()
	if signed {
		packager = compiler.NewZIPPackagerWithSigner(signer)
	}
	builder, err := compiler.NewGoApplicationExecutableBuilder(compiler.NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := compiler.NewRunnableManifestCompiler(builder, packager)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "staged-runnable.zip")
	if err := coordinator.Compile(context.Background(), compiler.RunnableManifestRequest{
		Admission:        admission,
		WorkingDirectory: repositoryRoot,
		OutputPath:       path,
	}); err != nil {
		t.Fatal(err)
	}

	runtimeEntrypoint := compiler.RuntimeEntrypoint{
		Module:  entrypoint.Module,
		Version: entrypoint.Version,
	}
	return runnableVerificationFixture{
		path: path,
		expected: runnablePackageExpectation{
			KeyID:           keyID,
			PublicKey:       append(ed25519.PublicKey(nil), privateKey.Public().(ed25519.PublicKey)...),
			ManifestName:    m.Name,
			ManifestVersion: m.Version,
			Entrypoint:      runtimeEntrypoint,
			TargetOS:        runtime.GOOS,
			TargetArch:      runtime.GOARCH,
			ImportPath:      runnableVerificationTestImportPath,
		},
	}
}

func runnableVerificationRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runnable verification test path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
	if _, err := os.Stat(filepath.Join(repositoryRoot, "go.mod")); err != nil {
		t.Fatalf("resolve repository root %q: %v", repositoryRoot, err)
	}
	return repositoryRoot
}

func requireRunnableVerificationError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}
}
