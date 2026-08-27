package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/internal/bootstrap"
	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/compiler"
)

const (
	buildRunnableTestModule     = "app"
	buildRunnableTestVersion    = "v1"
	buildRunnableTestImportPath = "example.com/forge-runnable-cli-test"
	buildRunnableTestKeyID      = "forge-runnable-test"
)

type buildRunnableFixture struct {
	directory    string
	manifestPath string
	keyPath      string
	privateKey   ed25519.PrivateKey
}

func TestBuildRunnableCommandCreatesDefaultSignedPackageV2(t *testing.T) {
	fixture := newBuildRunnableFixture(t, true, true)
	t.Chdir(fixture.directory)
	application := bootstrap.NewApplication()
	packages, sources := buildCommandRegistries(t, application)
	seedBuildCommandRegistries(t, packages, sources)
	before := snapshotBuildCommandRegistries(packages, sources)

	stdout, err := executeBuildRunnableCommand(
		application,
		nil,
		fixture.manifestPath,
		fixture.keyPath,
		buildRunnableTestKeyID,
		"",
	)
	if err != nil {
		t.Fatal(err)
	}

	finalPath := filepath.Join(
		fixture.directory,
		"build",
		"demo-v1-runnable-"+runtime.GOOS+"-"+runtime.GOARCH+".zip",
	)
	wantOutput := fmt.Sprintf(
		"Runnable package created: %s (format=2, bundle=2, entrypoint=app@v1, target=%s/%s, signer=%s)\n",
		finalPath,
		runtime.GOOS,
		runtime.GOARCH,
		buildRunnableTestKeyID,
	)
	if stdout != wantOutput {
		t.Fatalf("expected success output %q, got %q", wantOutput, stdout)
	}

	result := readBuildRunnablePackage(t, finalPath, fixture.privateKey, buildRunnableTestKeyID)
	requireBuildRunnablePackageV2(t, result)
	requireBuildCommandRegistrySnapshot(t, packages, sources, before)
	requireNoRunnableStagingDirectories(t, filepath.Dir(finalPath))
}

func TestBuildRunnableCommandCustomOutputPaths(t *testing.T) {
	for _, test := range []struct {
		name      string
		requested func(string) string
		expected  func(string) string
	}{
		{
			name: "relative",
			requested: func(string) string {
				return filepath.Join("dist", "custom.zip")
			},
			expected: func(directory string) string {
				return filepath.Join(directory, "dist", "custom.zip")
			},
		},
		{
			name: "absolute",
			requested: func(directory string) string {
				return filepath.Join(directory, "published", "absolute.zip")
			},
			expected: func(directory string) string {
				return filepath.Join(directory, "published", "absolute.zip")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newBuildRunnableFixture(t, true, true)
			t.Chdir(fixture.directory)
			requested := test.requested(fixture.directory)
			finalPath := test.expected(fixture.directory)

			stdout, err := executeBuildRunnableCommand(
				bootstrap.NewApplication(),
				nil,
				fixture.manifestPath,
				fixture.keyPath,
				buildRunnableTestKeyID,
				requested,
			)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout, finalPath) {
				t.Fatalf("expected absolute output path %q in %q", finalPath, stdout)
			}
			requireBuildRunnablePackageV2(
				t,
				readBuildRunnablePackage(t, finalPath, fixture.privateKey, buildRunnableTestKeyID),
			)
			defaultPath := filepath.Join(
				fixture.directory,
				"build",
				"demo-v1-runnable-"+runtime.GOOS+"-"+runtime.GOARCH+".zip",
			)
			if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
				t.Fatalf("custom output unexpectedly created default output: %v", err)
			}
			requireNoRunnableStagingDirectories(t, filepath.Dir(finalPath))
		})
	}
}

func TestBuildRunnableCommandRequiresEntrypointBeforeKeyLoad(t *testing.T) {
	fixture := newBuildRunnableFixture(t, true, false)
	t.Chdir(fixture.directory)
	outputPath := filepath.Join(fixture.directory, "missing-entrypoint.zip")

	_, err := executeBuildRunnableCommand(
		bootstrap.NewApplication(),
		nil,
		fixture.manifestPath,
		filepath.Join(fixture.directory, "does-not-exist.pem"),
		buildRunnableTestKeyID,
		outputPath,
	)
	if !errors.Is(err, compiler.ErrInvalidApplicationEntrypoint) {
		t.Fatalf("expected ErrInvalidApplicationEntrypoint, got %v", err)
	}
	if errors.Is(err, compiler.ErrInvalidPackageSignature) {
		t.Fatalf("signing key was consulted before entrypoint validation: %v", err)
	}
	requireBuildCommandOutputAbsent(t, outputPath)
	requireNoRunnableStagingDirectories(t, filepath.Dir(outputPath))
}

func TestBuildRunnableCommandSigningFailuresCreateNoOutput(t *testing.T) {
	tests := map[string]func(*testing.T, buildRunnableFixture) (string, string){
		"missing signing key": func(_ *testing.T, _ buildRunnableFixture) (string, string) {
			return "", buildRunnableTestKeyID
		},
		"missing key ID": func(_ *testing.T, fixture buildRunnableFixture) (string, string) {
			return fixture.keyPath, ""
		},
		"noncanonical key ID": func(_ *testing.T, fixture buildRunnableFixture) (string, string) {
			return fixture.keyPath, " " + buildRunnableTestKeyID
		},
		"malformed key": func(t *testing.T, fixture buildRunnableFixture) (string, string) {
			path := filepath.Join(fixture.directory, "malformed.pem")
			if err := os.WriteFile(path, []byte("not a PEM key"), 0o600); err != nil {
				t.Fatal(err)
			}
			return path, buildRunnableTestKeyID
		},
		"wrong key format": func(t *testing.T, fixture buildRunnableFixture) (string, string) {
			path := filepath.Join(fixture.directory, "raw.key")
			if err := os.WriteFile(path, fixture.privateKey, 0o600); err != nil {
				t.Fatal(err)
			}
			return path, buildRunnableTestKeyID
		},
	}

	for name, configure := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newBuildRunnableFixture(t, true, true)
			t.Chdir(fixture.directory)
			keyPath, keyID := configure(t, fixture)
			outputPath := filepath.Join(fixture.directory, "must-not-exist.zip")

			_, err := executeBuildRunnableCommand(
				bootstrap.NewApplication(),
				nil,
				fixture.manifestPath,
				keyPath,
				keyID,
				outputPath,
			)
			if !errors.Is(err, compiler.ErrInvalidPackageSignature) {
				t.Fatalf("expected ErrInvalidPackageSignature, got %v", err)
			}
			requireBuildCommandOutputAbsent(t, outputPath)
			requireNoRunnableStagingDirectories(t, filepath.Dir(outputPath))
		})
	}
}

func TestBuildRunnableCommandRejectsNonMainWithoutRegistryMutation(t *testing.T) {
	fixture := newBuildRunnableFixture(t, false, true)
	t.Chdir(fixture.directory)
	application := bootstrap.NewApplication()
	packages, sources := buildCommandRegistries(t, application)
	seedBuildCommandRegistries(t, packages, sources)
	before := snapshotBuildCommandRegistries(packages, sources)
	outputPath := filepath.Join(fixture.directory, "non-main.zip")

	_, err := executeBuildRunnableCommand(
		application,
		nil,
		fixture.manifestPath,
		fixture.keyPath,
		buildRunnableTestKeyID,
		outputPath,
	)
	if !errors.Is(err, compiler.ErrInvalidApplicationEntrypoint) {
		t.Fatalf("expected ErrInvalidApplicationEntrypoint, got %v", err)
	}
	requireBuildCommandOutputAbsent(t, outputPath)
	requireBuildCommandRegistrySnapshot(t, packages, sources, before)
	requireNoRunnableStagingDirectories(t, filepath.Dir(outputPath))
}

func TestBuildRunnableCommandNeverOverwritesExistingOutput(t *testing.T) {
	fixture := newBuildRunnableFixture(t, true, true)
	t.Chdir(fixture.directory)
	outputPath := filepath.Join(fixture.directory, "existing.zip")
	sentinel := []byte("existing output must remain unchanged")
	if err := os.WriteFile(outputPath, sentinel, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := executeBuildRunnableCommand(
		bootstrap.NewApplication(),
		nil,
		fixture.manifestPath,
		fixture.keyPath,
		buildRunnableTestKeyID,
		outputPath,
	)
	if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}
	got, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, sentinel) {
		t.Fatalf("existing output changed: got %q", got)
	}
	requireNoRunnableStagingDirectories(t, filepath.Dir(outputPath))
}

func TestBuildRunnableCommandPreCanceledContextHasNoSideEffects(t *testing.T) {
	fixture := newBuildRunnableFixture(t, true, true)
	t.Chdir(fixture.directory)
	application := bootstrap.NewApplication()
	packages, sources := buildCommandRegistries(t, application)
	before := snapshotBuildCommandRegistries(packages, sources)
	outputPath := filepath.Join(fixture.directory, "canceled.zip")
	ctx, cancel := context.WithCancel(NewApplicationContext(application))
	cancel()

	_, err := executeBuildRunnableCommand(
		application,
		ctx,
		fixture.manifestPath,
		fixture.keyPath,
		buildRunnableTestKeyID,
		outputPath,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	requireBuildCommandOutputAbsent(t, outputPath)
	requireBuildCommandRegistrySnapshot(t, packages, sources, before)
	requireNoRunnableStagingDirectories(t, filepath.Dir(outputPath))
}

func TestBuildRunnableCommandHelpAndArguments(t *testing.T) {
	t.Run("help", func(t *testing.T) {
		cmd := NewRootCommand()
		var output strings.Builder
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		cmd.SetArgs([]string{"build-runnable", "--help"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		for _, expected := range []string{
			"build-runnable",
			"--signing-key",
			"--key-id",
			"--output",
		} {
			if !strings.Contains(output.String(), expected) {
				t.Fatalf("expected help to contain %q, got %q", expected, output.String())
			}
		}
	})

	for name, args := range map[string][]string{
		"zero manifest arguments": {"build-runnable"},
		"multiple manifest arguments": {
			"build-runnable",
			"first.yaml",
			"second.yaml",
		},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := NewRootCommand()
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected exact argument validation failure")
			}
		})
	}
}

func executeBuildRunnableCommand(
	application *app.App,
	ctx context.Context,
	manifestPath,
	keyPath,
	keyID,
	outputPath string,
) (string, error) {
	cmd := NewRootCommandWithApplication(application)
	if ctx != nil {
		cmd.SetContext(ctx)
	}
	args := []string{"build-runnable", manifestPath}
	if keyPath != "" {
		args = append(args, "--signing-key", keyPath)
	}
	if keyID != "" {
		args = append(args, "--key-id", keyID)
	}
	if outputPath != "" {
		args = append(args, "--output", outputPath)
	}
	cmd.SetArgs(args)
	var output strings.Builder
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	err := cmd.Execute()
	return output.String(), err
}

func newBuildRunnableFixture(
	t *testing.T,
	mainPackage,
	withEntrypoint bool,
) buildRunnableFixture {
	t.Helper()
	t.Setenv("GOTELEMETRY", "off")
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module "+buildRunnableTestImportPath+"\n\ngo 1.26\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	packageName := "library"
	if mainPackage {
		packageName = "main"
	}
	if err := os.WriteFile(
		filepath.Join(directory, "application.go"),
		[]byte("package "+packageName+"\n\nfunc main() {}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	entrypoint := ""
	if withEntrypoint {
		entrypoint = "entrypoint:\n  module: app\n  version: v1\n"
	}
	manifestData := fmt.Sprintf(
		"version: v1\nname: demo\n%smodules:\n  - name: app\n    version: v1\n    import_path: \" %s \"\n",
		entrypoint,
		buildRunnableTestImportPath,
	)
	manifestPath := filepath.Join(directory, "forge.yaml")
	if err := os.WriteFile(manifestPath, []byte(manifestData), 0o644); err != nil {
		t.Fatal(err)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(directory, "signing-key.pem")
	if err := os.WriteFile(keyPath, validRunnableSigningKeyPEM(t, privateKey), 0o600); err != nil {
		t.Fatal(err)
	}

	return buildRunnableFixture{
		directory:    directory,
		manifestPath: manifestPath,
		keyPath:      keyPath,
		privateKey:   append(ed25519.PrivateKey(nil), privateKey...),
	}
}

func readBuildRunnablePackage(
	t *testing.T,
	path string,
	privateKey ed25519.PrivateKey,
	keyID string,
) compiler.PackageReadResult {
	t.Helper()
	trustStore := compiler.NewTrustStore()
	if err := trustStore.Register(keyID, privateKey.Public().(ed25519.PublicKey)); err != nil {
		t.Fatal(err)
	}
	verifier, err := compiler.NewEd25519VerifierWithTrustStore(trustStore)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := compiler.NewZIPPackageReaderWithPolicyAndVerifierAndLimits(
		compiler.StrictPackageVerificationPolicy(),
		verifier,
		compiler.AlphaRuntimePackageReadLimits(),
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := reader.ReadDetailed(path)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func requireBuildRunnablePackageV2(t *testing.T, result compiler.PackageReadResult) {
	t.Helper()
	if result.PackageFormatVersion != 2 || result.BundleSchemaVersion != 2 {
		t.Fatalf(
			"expected format/schema 2, got %d/%d",
			result.PackageFormatVersion,
			result.BundleSchemaVersion,
		)
	}
	if result.VerifiedSignerKeyID != buildRunnableTestKeyID {
		t.Fatalf("unexpected signer %q", result.VerifiedSignerKeyID)
	}
	if result.Bundle.ManifestName != "demo" || result.Bundle.ManifestVersion != "v1" {
		t.Fatalf("unexpected manifest identity %#v", result.Bundle)
	}
	if result.Bundle.Runtime == nil {
		t.Fatal("expected runtime descriptor")
	}
	wantEntrypoint := compiler.RuntimeEntrypoint{
		Module:  buildRunnableTestModule,
		Version: buildRunnableTestVersion,
	}
	if result.Bundle.Runtime.Kind != compiler.RuntimeKindApplicationExecutable ||
		result.Bundle.Runtime.Entrypoint != wantEntrypoint ||
		result.Bundle.Runtime.TargetOS != runtime.GOOS ||
		result.Bundle.Runtime.TargetArch != runtime.GOARCH {
		t.Fatalf("unexpected runtime descriptor %#v", result.Bundle.Runtime)
	}
	if len(result.Bundle.Artifacts) != 1 {
		t.Fatalf("expected one artifact, got %d", len(result.Bundle.Artifacts))
	}
	artifact := result.Bundle.Artifacts[0]
	if artifact.Module != buildRunnableTestModule ||
		artifact.Version != buildRunnableTestVersion ||
		artifact.ImportPath != buildRunnableTestImportPath {
		t.Fatalf("unexpected artifact %#v", artifact)
	}
	payload, present := result.Payloads[buildRunnableTestModule+"@"+buildRunnableTestVersion]
	if len(result.Payloads) != 1 || !present || len(payload) == 0 {
		t.Fatal("expected one non-empty executable payload")
	}
}

func requireNoRunnableStagingDirectories(t *testing.T, parent string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(parent, ".forge-runnable-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("staging directories leaked: %#v", matches)
	}
}
