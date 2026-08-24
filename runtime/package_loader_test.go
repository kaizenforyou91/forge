package runtime

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/kaizenforyou91/forge/pkg/manifest"
)

const (
	testSignerKeyID = "runtime-loader-test"
	testModule      = "runnable-app"
	testVersion     = "v1"
	testImportPath  = "github.com/kaizenforyou91/forge/pkg/compiler/testdata/runnable_app"
)

type runnablePackageFixture struct {
	packagePath string
	executable  []byte
	signer      *compiler.Ed25519Signer
	trustStore  *compiler.TrustStore
}

type bundleV2Wire struct {
	ManifestName    string         `json:"manifest_name"`
	ManifestVersion string         `json:"manifest_version"`
	Runtime         runtimeV2Wire  `json:"runtime"`
	Artifacts       []artifactWire `json:"artifacts"`
}

type runtimeV2Wire struct {
	Kind       string         `json:"kind"`
	Entrypoint entrypointWire `json:"entrypoint"`
	TargetOS   string         `json:"target_os"`
	TargetArch string         `json:"target_arch"`
}

type entrypointWire struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

type artifactWire struct {
	Module     string `json:"module"`
	Version    string `json:"version"`
	ImportPath string `json:"import_path"`
}

func TestNewVerifiedRunnablePackageLoaderRequiresTrustStore(t *testing.T) {
	loader, err := NewVerifiedRunnablePackageLoader(nil)
	if !errors.Is(err, compiler.ErrNilTrustStore) {
		t.Fatalf("expected compiler.ErrNilTrustStore, got %v", err)
	}
	if loader != nil {
		t.Fatal("expected nil loader")
	}
}

func TestVerifiedRunnablePackageLoaderValidatesInput(t *testing.T) {
	var nilLoader *VerifiedRunnablePackageLoader
	if _, err := nilLoader.Load("package.zip"); !errors.Is(err, ErrInvalidRunnablePackage) {
		t.Fatalf("expected ErrInvalidRunnablePackage for nil loader, got %v", err)
	}

	loader := newTestLoader(t, compiler.NewTrustStore())
	if _, err := loader.Load("  "); !errors.Is(err, ErrInvalidRunnablePackage) {
		t.Fatalf("expected ErrInvalidRunnablePackage for blank path, got %v", err)
	}
}

func TestVerifiedRunnablePackageLoaderLoadsTrustedSignedV2(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)
	loader := newTestLoader(t, fixture.trustStore)

	result, err := loader.Load(fixture.packagePath)
	if err != nil {
		t.Fatal(err)
	}

	if result.PackageFormatVersion() != 2 {
		t.Fatalf("expected package format 2, got %d", result.PackageFormatVersion())
	}
	if result.BundleSchemaVersion() != 2 {
		t.Fatalf("expected bundle schema 2, got %d", result.BundleSchemaVersion())
	}
	if result.ManifestName() != "runtime-loader-fixture" || result.ManifestVersion() != "v1" {
		t.Fatalf("unexpected manifest identity %s@%s", result.ManifestName(), result.ManifestVersion())
	}
	if result.Entrypoint() != (compiler.RuntimeEntrypoint{Module: testModule, Version: testVersion}) {
		t.Fatalf("unexpected entrypoint %#v", result.Entrypoint())
	}
	if result.ImportPath() != testImportPath {
		t.Fatalf("unexpected import path %q", result.ImportPath())
	}
	if result.TargetOS() != goruntime.GOOS || result.TargetArch() != goruntime.GOARCH {
		t.Fatalf("unexpected target %s/%s", result.TargetOS(), result.TargetArch())
	}
	if result.SignerKeyID() != testSignerKeyID {
		t.Fatalf("unexpected signer key ID %q", result.SignerKeyID())
	}
	if !bytes.Equal(result.ExecutableBytes(), fixture.executable) {
		t.Fatal("loaded executable bytes differ from packaged executable")
	}
}

func TestVerifiedRunnablePackageLoaderRejectsTrustedSignedV1AsNotRunnable(t *testing.T) {
	signer, trustStore := newTestSignerAndTrustStore(t)
	packagePath := filepath.Join(t.TempDir(), "signed-v1.zip")
	bundle := compiler.ArtifactBundle{
		ManifestName:    "inspection-package",
		ManifestVersion: "v1",
		Artifacts: []compiler.Artifact{{
			Module:     "library",
			Version:    "v1",
			ImportPath: "example.com/library",
		}},
	}
	if err := compiler.NewZIPPackagerWithSigner(signer).Package(
		bundle,
		map[string][]byte{"library@v1": []byte("library@v1")},
		packagePath,
	); err != nil {
		t.Fatal(err)
	}

	_, err := newTestLoader(t, trustStore).Load(packagePath)
	if !errors.Is(err, ErrPackageNotRunnable) {
		t.Fatalf("expected ErrPackageNotRunnable, got %v", err)
	}
}

func TestVerifiedRunnablePackageLoaderPreservesSignatureAndTrustFailures(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)

	t.Run("unsigned", func(t *testing.T) {
		entries := readZIPEntries(t, fixture.packagePath)
		delete(entries, "signature.json")
		path := writeZIPEntries(t, entries, zip.Store)

		_, err := newTestLoader(t, fixture.trustStore).Load(path)
		if !errors.Is(err, compiler.ErrMissingPackageSignature) {
			t.Fatalf("expected ErrMissingPackageSignature, got %v", err)
		}
	})

	t.Run("untrusted key", func(t *testing.T) {
		_, err := newTestLoader(t, compiler.NewTrustStore()).Load(fixture.packagePath)
		if !errors.Is(err, compiler.ErrUntrustedPackageKey) {
			t.Fatalf("expected ErrUntrustedPackageKey, got %v", err)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		entries := readZIPEntries(t, fixture.packagePath)
		signature, err := compiler.UnmarshalPackageSignature(entries["signature.json"])
		if err != nil {
			t.Fatal(err)
		}
		raw, err := base64.StdEncoding.DecodeString(signature.Signature)
		if err != nil {
			t.Fatal(err)
		}
		raw[0] ^= 0xff
		signature.Signature = base64.StdEncoding.EncodeToString(raw)
		entries["signature.json"], err = compiler.MarshalPackageSignature(signature)
		if err != nil {
			t.Fatal(err)
		}
		path := writeZIPEntries(t, entries, zip.Store)

		_, err = newTestLoader(t, fixture.trustStore).Load(path)
		if !errors.Is(err, compiler.ErrSignatureMismatch) {
			t.Fatalf("expected ErrSignatureMismatch, got %v", err)
		}
	})
}

func TestVerifiedRunnablePackageLoaderPreservesIntegrityFailures(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)

	t.Run("payload tamper", func(t *testing.T) {
		entries := readZIPEntries(t, fixture.packagePath)
		payloadPath := artifactArchivePath(testModule, testVersion)
		entries[payloadPath][0] ^= 0xff
		path := writeZIPEntries(t, entries, zip.Store)

		_, err := newTestLoader(t, fixture.trustStore).Load(path)
		if !errors.Is(err, compiler.ErrIntegrityMismatch) {
			t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
		}
	})

	t.Run("runtime metadata tamper", func(t *testing.T) {
		entries := readZIPEntries(t, fixture.packagePath)
		bundle := unmarshalBundleV2(t, entries["bundle.json"])
		bundle.Runtime.TargetOS = "tampered-" + bundle.Runtime.TargetOS
		entries["bundle.json"] = marshalBundleV2(t, bundle)
		path := writeZIPEntries(t, entries, zip.Store)

		_, err := newTestLoader(t, fixture.trustStore).Load(path)
		if !errors.Is(err, compiler.ErrIntegrityMismatch) {
			t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
		}
	})
}

func TestVerifiedRunnablePackageLoaderRejectsUnsupportedPlatform(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)

	for name, mutate := range map[string]func(*bundleV2Wire){
		"target OS": func(bundle *bundleV2Wire) {
			bundle.Runtime.TargetOS = "forge-unsupported-os"
		},
		"target architecture": func(bundle *bundleV2Wire) {
			bundle.Runtime.TargetArch = "forge-unsupported-arch"
		},
		"case mismatch": func(bundle *bundleV2Wire) {
			bundle.Runtime.TargetOS = strings.ToUpper(goruntime.GOOS)
		},
	} {
		t.Run(name, func(t *testing.T) {
			entries := readZIPEntries(t, fixture.packagePath)
			bundle := unmarshalBundleV2(t, entries["bundle.json"])
			mutate(&bundle)
			entries["bundle.json"] = marshalBundleV2(t, bundle)
			resignPackageEntries(t, entries, fixture.signer)
			packagePath := writeZIPEntries(t, entries, zip.Store)

			_, err := newTestLoader(t, fixture.trustStore).Load(packagePath)
			if !errors.Is(err, ErrUnsupportedRuntimePlatform) {
				t.Fatalf("expected ErrUnsupportedRuntimePlatform, got %v", err)
			}
		})
	}
}

func TestVerifiedRunnablePackageLoaderRejectsMultipleArtifacts(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)
	entries := readZIPEntries(t, fixture.packagePath)
	bundle := unmarshalBundleV2(t, entries["bundle.json"])
	bundle.Artifacts = append(bundle.Artifacts, artifactWire{
		Module:     "dependency",
		Version:    "v1",
		ImportPath: "example.com/dependency",
	})
	entries["bundle.json"] = marshalBundleV2(t, bundle)
	entries[artifactArchivePath("dependency", "v1")] = []byte("statically-linked-provenance")
	resignPackageEntries(t, entries, fixture.signer)
	packagePath := writeZIPEntries(t, entries, zip.Store)

	_, err := newTestLoader(t, fixture.trustStore).Load(packagePath)
	if !errors.Is(err, ErrInvalidRunnablePackage) {
		t.Fatalf("expected ErrInvalidRunnablePackage, got %v", err)
	}
}

func TestVerifiedRunnablePackageLoaderRejectsEmptyExecutablePayload(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)
	entries := readZIPEntries(t, fixture.packagePath)
	entries[artifactArchivePath(testModule, testVersion)] = []byte{}
	resignPackageEntries(t, entries, fixture.signer)
	packagePath := writeZIPEntries(t, entries, zip.Store)

	_, err := newTestLoader(t, fixture.trustStore).Load(packagePath)
	if !errors.Is(err, ErrInvalidRunnablePackage) {
		t.Fatalf("expected ErrInvalidRunnablePackage, got %v", err)
	}
}

func TestVerifiedRunnablePackageLoaderRequiresVerifiedSignerEvidence(t *testing.T) {
	_, err := verifiedRunnablePackageFromReadResult(compiler.PackageReadResult{
		PackageFormatVersion: 2,
		BundleSchemaVersion:  2,
	})
	if !errors.Is(err, ErrInvalidRunnablePackage) {
		t.Fatalf("expected ErrInvalidRunnablePackage, got %v", err)
	}
}

func TestVerifiedRunnablePackageExecutableBytesAreDetached(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)
	result, err := newTestLoader(t, fixture.trustStore).Load(fixture.packagePath)
	if err != nil {
		t.Fatal(err)
	}

	first := result.ExecutableBytes()
	first[0] ^= 0xff
	second := result.ExecutableBytes()
	if !bytes.Equal(second, fixture.executable) {
		t.Fatal("ExecutableBytes returned aliased mutable storage")
	}

	if err := os.WriteFile(fixture.packagePath, []byte("source replaced after load"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(result.ExecutableBytes(), fixture.executable) {
		t.Fatal("loaded bytes changed after source archive replacement")
	}
}

func TestVerifiedRunnablePackageLoaderDoesNotMutatePackageSource(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)
	beforeBytes, err := os.ReadFile(fixture.packagePath)
	if err != nil {
		t.Fatal(err)
	}
	beforeFiles := directoryFiles(t, filepath.Dir(fixture.packagePath))

	if _, err := newTestLoader(t, fixture.trustStore).Load(fixture.packagePath); err != nil {
		t.Fatal(err)
	}

	afterBytes, err := os.ReadFile(fixture.packagePath)
	if err != nil {
		t.Fatal(err)
	}
	afterFiles := directoryFiles(t, filepath.Dir(fixture.packagePath))
	if !bytes.Equal(afterBytes, beforeBytes) {
		t.Fatal("loader mutated package bytes")
	}
	if !equalStrings(beforeFiles, afterFiles) {
		t.Fatalf("loader changed package directory contents: before=%v after=%v", beforeFiles, afterFiles)
	}
}

func TestVerifiedRunnablePackageLoaderUsesAlphaStoreOnlyPolicy(t *testing.T) {
	fixture := buildSignedRunnablePackageFixture(t)
	deflatedPath := writeZIPEntries(t, readZIPEntries(t, fixture.packagePath), zip.Deflate)

	verifier, err := compiler.NewEd25519VerifierWithTrustStore(fixture.trustStore)
	if err != nil {
		t.Fatal(err)
	}
	inspectionReader, err := compiler.NewZIPPackageReaderWithPolicyAndVerifier(
		compiler.StrictPackageVerificationPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := inspectionReader.ReadDetailed(deflatedPath); err != nil {
		t.Fatalf("generic strict reader should accept Deflate fixture: %v", err)
	}

	_, err = newTestLoader(t, fixture.trustStore).Load(deflatedPath)
	if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage from Store-only loader, got %v", err)
	}
}

func buildSignedRunnablePackageFixture(t *testing.T) runnablePackageFixture {
	t.Helper()

	signer, trustStore := newTestSignerAndTrustStore(t)
	sources := compiler.NewPackageSourceRegistry()
	if err := sources.Register(compiler.PackageSource{
		Name:       testModule,
		Version:    testVersion,
		ImportPath: testImportPath,
	}); err != nil {
		t.Fatal(err)
	}
	builder, err := compiler.NewGoApplicationExecutableBuilder(compiler.NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := compiler.NewRunnablePackageCompiler(
		sources,
		builder,
		compiler.NewZIPPackagerWithSigner(signer),
	)
	if err != nil {
		t.Fatal(err)
	}

	packagePath := filepath.Join(t.TempDir(), "runnable-v2.zip")
	request := compiler.RunnablePackageRequest{
		Plan: manifest.BuildPlan{
			ManifestName:    "runtime-loader-fixture",
			ManifestVersion: "v1",
			Steps: []manifest.BuildStep{{
				Module: testModule + "@" + testVersion,
			}},
		},
		Entrypoint: compiler.RuntimeEntrypoint{
			Module:  testModule,
			Version: testVersion,
		},
		WorkingDirectory: repositoryRoot(t),
		OutputPath:       packagePath,
	}
	if err := coordinator.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntries(t, packagePath)
	executable := append([]byte(nil), entries[artifactArchivePath(testModule, testVersion)]...)
	if len(executable) == 0 {
		t.Fatal("runnable fixture contains an empty executable")
	}

	return runnablePackageFixture{
		packagePath: packagePath,
		executable:  executable,
		signer:      signer,
		trustStore:  trustStore,
	}
}

func newTestSignerAndTrustStore(t *testing.T) (*compiler.Ed25519Signer, *compiler.TrustStore) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := compiler.NewEd25519Signer(testSignerKeyID, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	trustStore := compiler.NewTrustStore()
	if err := trustStore.Register(testSignerKeyID, publicKey); err != nil {
		t.Fatal(err)
	}

	return signer, trustStore
}

func newTestLoader(t *testing.T, trustStore *compiler.TrustStore) *VerifiedRunnablePackageLoader {
	t.Helper()

	loader, err := NewVerifiedRunnablePackageLoader(trustStore)
	if err != nil {
		t.Fatal(err)
	}
	return loader
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := goruntime.Caller(0)
	if !ok {
		t.Fatal("resolve runtime test source path")
	}
	return filepath.Dir(filepath.Dir(filename))
}

func artifactArchivePath(module, version string) string {
	return path.Join("artifacts", module, version, "artifact")
}

func readZIPEntries(t *testing.T, packagePath string) map[string][]byte {
	t.Helper()

	reader, err := zip.OpenReader(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	entries := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		entryReader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(entryReader)
		closeErr := entryReader.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		entries[file.Name] = append([]byte(nil), data...)
	}
	return entries
}

func writeZIPEntries(t *testing.T, entries map[string][]byte, method uint16) string {
	t.Helper()

	packagePath := filepath.Join(t.TempDir(), "fixture.zip")
	file, err := os.Create(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry, err := writer.CreateHeader(&zip.FileHeader{
			Name:     name,
			Method:   method,
			Modified: time.Unix(0, 0).UTC(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(entries[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return packagePath
}

func unmarshalBundleV2(t *testing.T, data []byte) bundleV2Wire {
	t.Helper()

	var bundle bundleV2Wire
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func marshalBundleV2(t *testing.T, bundle bundleV2Wire) []byte {
	t.Helper()

	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func resignPackageEntries(
	t *testing.T,
	entries map[string][]byte,
	signer *compiler.Ed25519Signer,
) {
	t.Helper()

	bundle := unmarshalBundleV2(t, entries["bundle.json"])
	integrity := compiler.PackageIntegrity{
		Version:               2,
		Algorithm:             "sha256",
		PackageMetadataSHA256: sha256Hex(entries["package.json"]),
		BundleSHA256:          sha256Hex(entries["bundle.json"]),
		Artifacts:             make([]compiler.ArtifactDigest, 0, len(bundle.Artifacts)),
	}
	for _, artifact := range bundle.Artifacts {
		payload := entries[artifactArchivePath(artifact.Module, artifact.Version)]
		integrity.Artifacts = append(integrity.Artifacts, compiler.ArtifactDigest{
			Module:  artifact.Module,
			Version: artifact.Version,
			SHA256:  sha256Hex(payload),
		})
	}

	integrityJSON, err := compiler.MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := signer.Sign(integrityJSON)
	if err != nil {
		t.Fatal(err)
	}
	signatureJSON, err := compiler.MarshalPackageSignature(signature)
	if err != nil {
		t.Fatal(err)
	}
	entries["integrity.json"] = integrityJSON
	entries["signature.json"] = signatureJSON
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func directoryFiles(t *testing.T, directory string) []string {
	t.Helper()

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
