package cli

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestInspectCommandPrintsExactV1UnsignedReportInBundleOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multi.zip")
	bundle := compiler.ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []compiler.Artifact{
			{Module: "web", Version: "v2", ImportPath: "example.com/web"},
			{Module: "core", Version: "v1", ImportPath: "example.com/core"},
		},
	}
	if err := compiler.NewZIPPackager().Package(
		bundle,
		map[string][]byte{
			"web@v2":  []byte("web"),
			"core@v1": []byte("core"),
		},
		path,
	); err != nil {
		t.Fatal(err)
	}

	stdout, err := executeInspectCommand(t, "inspect", path)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(
		"Package: %s\n"+
			"Format: 1\n"+
			"Bundle schema: 1\n"+
			"Manifest: demo@v1\n"+
			"Type: non-runnable\n"+
			"Runtime: none\n"+
			"Integrity: verified\n"+
			"Signature: unsigned\n"+
			"Artifacts: 2\n"+
			"  - web@v2 - example.com/web\n"+
			"  - core@v1 - example.com/core\n",
		filepath.Clean(path),
	)
	if stdout != want {
		t.Fatalf("unexpected report:\n%s\nwant:\n%s", stdout, want)
	}
}

func TestInspectCommandPrintsExactV2UnverifiedAndTrustedReports(t *testing.T) {
	fixture := newInspectV2Fixture(t)

	unverified, err := executeInspectCommand(t, "inspect", fixture.packagePath)
	if err != nil {
		t.Fatal(err)
	}
	wantUnverified := fmt.Sprintf(
		"Package: %s\n"+
			"Format: 2\n"+
			"Bundle schema: 2\n"+
			"Manifest: demo@v2\n"+
			"Type: runnable\n"+
			"Runtime: application_executable\n"+
			"Target: %s/amd64\n"+
			"Entrypoint: app@v1\n"+
			"Integrity: verified\n"+
			"Signature: present, trust not verified\n"+
			"Declared KeyID (unverified): %s\n"+
			"Artifacts: 1\n"+
			"  - app@v1 - example.com/demo/app\n",
		filepath.Clean(fixture.packagePath),
		fixture.targetOS,
		fixture.keyID,
	)
	if unverified != wantUnverified {
		t.Fatalf("unexpected unverified report:\n%s\nwant:\n%s", unverified, wantUnverified)
	}
	for _, forbidden := range []string{"Trusted signer", "Authentic", "Verified signer", "Trusted package"} {
		if strings.Contains(unverified, forbidden) {
			t.Fatalf("unverified report contains forbidden authenticity wording %q", forbidden)
		}
	}

	trusted, err := executeInspectCommand(
		t,
		"inspect",
		fixture.packagePath,
		"--trusted-key",
		fixture.keyPath,
		"--key-id",
		fixture.keyID,
	)
	if err != nil {
		t.Fatal(err)
	}
	wantTrusted := fmt.Sprintf(
		"Package: %s\n"+
			"Format: 2\n"+
			"Bundle schema: 2\n"+
			"Manifest: demo@v2\n"+
			"Type: runnable\n"+
			"Runtime: application_executable\n"+
			"Target: %s/amd64\n"+
			"Entrypoint: app@v1\n"+
			"Integrity: verified\n"+
			"Signature: trusted\n"+
			"Verified signer: %s\n"+
			"Artifacts: 1\n"+
			"  - app@v1 - example.com/demo/app\n",
		filepath.Clean(fixture.packagePath),
		fixture.targetOS,
		fixture.keyID,
	)
	if trusted != wantTrusted {
		t.Fatalf("unexpected trusted report:\n%s\nwant:\n%s", trusted, wantTrusted)
	}
	if strings.Contains(trusted, "Declared KeyID (unverified)") {
		t.Fatalf("trusted report retained unverified label: %q", trusted)
	}
}

func TestInspectCommandEnforcesGrammarAndTrustFlagPairing(t *testing.T) {
	fixture := newInspectV2Fixture(t)
	tests := map[string][]string{
		"missing package":        {"inspect"},
		"extra package":          {"inspect", fixture.packagePath, "extra.zip"},
		"trusted key only":       {"inspect", fixture.packagePath, "--trusted-key", fixture.keyPath},
		"key ID only":            {"inspect", fixture.packagePath, "--key-id", fixture.keyID},
		"empty trusted key":      {"inspect", fixture.packagePath, "--trusted-key=", "--key-id", fixture.keyID},
		"empty key ID":           {"inspect", fixture.packagePath, "--trusted-key", fixture.keyPath, "--key-id="},
		"whitespace key ID":      {"inspect", fixture.packagePath, "--trusted-key", fixture.keyPath, "--key-id", " " + fixture.keyID},
		"uppercase extension":    {"inspect", strings.TrimSuffix(fixture.packagePath, ".zip") + ".ZIP"},
		"URL input":              {"inspect", "https://example.com/app.zip"},
		"surrounding whitespace": {"inspect", " " + fixture.packagePath},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			stdout, err := executeInspectCommand(t, args...)
			if err == nil || ExitCode(err) != 1 {
				t.Fatalf("expected exit 1, got output %q, error %v", stdout, err)
			}
			if stdout != "" {
				t.Fatalf("failure printed partial success report %q", stdout)
			}
		})
	}
}

func TestInspectCommandTrustedModeRejectsUnsignedAndWrongTrust(t *testing.T) {
	fixture := newInspectV2Fixture(t)
	unsignedPath := filepath.Join(t.TempDir(), "unsigned.zip")
	if err := compiler.NewZIPPackager().Package(
		compiler.ArtifactBundle{
			ManifestName:    "unsigned",
			ManifestVersion: "v1",
			Artifacts: []compiler.Artifact{{
				Module: "app", Version: "v1", ImportPath: "example.com/app",
			}},
		},
		map[string][]byte{"app@v1": []byte("app")},
		unsignedPath,
	); err != nil {
		t.Fatal(err)
	}

	_, err := executeInspectCommand(
		t,
		"inspect",
		unsignedPath,
		"--trusted-key",
		fixture.keyPath,
		"--key-id",
		fixture.keyID,
	)
	if !errors.Is(err, compiler.ErrMissingPackageSignature) {
		t.Fatalf("expected ErrMissingPackageSignature, got %v", err)
	}

	_, err = executeInspectCommand(
		t,
		"inspect",
		fixture.packagePath,
		"--trusted-key",
		fixture.keyPath,
		"--key-id",
		"wrong-key-id",
	)
	if !errors.Is(err, compiler.ErrUntrustedPackageKey) {
		t.Fatalf("expected ErrUntrustedPackageKey, got %v", err)
	}

	wrongPublicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	wrongKeyPath := writeRunTrustedKeyFile(
		t,
		"wrong.pem",
		validRunTrustedKeyPEM(t, wrongPublicKey),
	)
	_, err = executeInspectCommand(
		t,
		"inspect",
		fixture.packagePath,
		"--trusted-key",
		wrongKeyPath,
		"--key-id",
		fixture.keyID,
	)
	if !errors.Is(err, compiler.ErrUntrustedPackageKey) {
		t.Fatalf("expected ErrUntrustedPackageKey, got %v", err)
	}
}

func TestInspectCommandDelegatesFilesystemValidationToReader(t *testing.T) {
	for name, path := range map[string]string{
		"missing":   filepath.Join(t.TempDir(), "missing.zip"),
		"directory": filepath.Join(t.TempDir(), "directory.zip"),
	} {
		t.Run(name, func(t *testing.T) {
			if name == "directory" {
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			stdout, err := executeInspectCommand(t, "inspect", path)
			if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
				t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
			}
			if stdout != "" {
				t.Fatalf("failure printed partial report %q", stdout)
			}
		})
	}
}

func TestInspectCommandPropagatesOutputWriterFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.zip")
	if err := compiler.NewZIPPackager().Package(
		compiler.ArtifactBundle{
			ManifestName:    "demo",
			ManifestVersion: "v1",
			Artifacts: []compiler.Artifact{{
				Module: "app", Version: "v1", ImportPath: "example.com/app",
			}},
		},
		map[string][]byte{"app@v1": []byte("app")},
		path,
	); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	cmd.SetOut(inspectFailingWriter{})
	cmd.SetArgs([]string{"inspect", path})
	if err := cmd.Execute(); err == nil || !strings.Contains(
		err.Error(),
		"write package inspection report",
	) {
		t.Fatalf("expected report writer failure, got %v", err)
	}
}

func TestInspectCommandHelpExposesOnlyAlphaFlags(t *testing.T) {
	stdout, err := executeInspectCommand(t, "inspect", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"inspect <package.zip>", "--trusted-key", "--key-id"} {
		if !strings.Contains(stdout, required) {
			t.Fatalf("inspect help lacks %q: %q", required, stdout)
		}
	}
	for _, forbidden := range []string{
		"--json", "--extract", "--execute", "--unsafe", "--skip-integrity",
		"--ignore-signature", "--host", "--platform", "--all",
	} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("inspect help exposes unsupported flag %q: %q", forbidden, stdout)
		}
	}
}

type inspectV2Fixture struct {
	packagePath string
	keyPath     string
	keyID       string
	targetOS    string
}

func newInspectV2Fixture(t *testing.T) inspectV2Fixture {
	t.Helper()
	const keyID = "inspect-release"
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := compiler.NewEd25519Signer(keyID, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	targetOS := "windows"
	if runtime.GOOS == targetOS {
		targetOS = "linux"
	}
	path := filepath.Join(t.TempDir(), "application.zip")
	writeInspectV2Package(t, path, targetOS, signer)
	return inspectV2Fixture{
		packagePath: path,
		keyPath: writeRunTrustedKeyFile(
			t,
			"public.pem",
			validRunTrustedKeyPEM(t, publicKey),
		),
		keyID:    keyID,
		targetOS: targetOS,
	}
}

func writeInspectV2Package(
	t *testing.T,
	path string,
	targetOS string,
	signer compiler.PackageSigner,
) {
	t.Helper()
	packageJSON := []byte(`{"package_format_version":2,"bundle_schema_version":2}`)
	bundleJSON := []byte(fmt.Sprintf(
		`{"manifest_name":"demo","manifest_version":"v2","runtime":{"kind":"application_executable","entrypoint":{"module":"app","version":"v1"},"target_os":%q,"target_arch":"amd64"},"artifacts":[{"module":"app","version":"v1","import_path":"example.com/demo/app"}]}`,
		targetOS,
	))
	payload := []byte("cross-host executable payload")
	integrityJSON, err := compiler.MarshalPackageIntegrity(compiler.PackageIntegrity{
		Version:               2,
		Algorithm:             "sha256",
		PackageMetadataSHA256: inspectSHA256(packageJSON),
		BundleSHA256:          inspectSHA256(bundleJSON),
		Artifacts: []compiler.ArtifactDigest{{
			Module: "app", Version: "v1", SHA256: inspectSHA256(payload),
		}},
	})
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
	writeInspectZIP(t, path, map[string][]byte{
		"package.json":              packageJSON,
		"bundle.json":               bundleJSON,
		"integrity.json":            integrityJSON,
		"signature.json":            signatureJSON,
		"artifacts/app/v1/artifact": payload,
	})
}

func writeInspectZIP(t *testing.T, path string, entries map[string][]byte) {
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
		entry, err := writer.CreateHeader(&zip.FileHeader{Name: key, Method: zip.Store})
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

func inspectSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func executeInspectCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

type inspectFailingWriter struct{}

func (inspectFailingWriter) Write([]byte) (int, error) {
	return 0, errors.New("injected inspect output failure")
}
