package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/internal/bootstrap"
	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/kaizenforyou91/forge/pkg/manifest"
)

const runTestKeyID = "forge-run-test"

type runCommandFixture struct {
	packagePath string
	keyPath     string
	publicKey   ed25519.PublicKey
}

func TestRunCommandRequiresExactGrammar(t *testing.T) {
	for _, args := range [][]string{{"run"}, {"run", "app.zip"}, {"run", "app.zip", "extra", "--trusted-key", "key.pem", "--key-id", "id"}} {
		cmd := NewRootCommand()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("expected grammar failure for %q", args)
		}
	}
}

func TestRunCommandExecutesTrustedPackageAndRoutesOutput(t *testing.T) {
	fixture := newRunCommandFixture(t, "process_success", true)
	before := append([]byte(nil), mustReadRunFile(t, fixture.packagePath)...)
	stdout, stderr, err := executeRunCommand(t, context.Background(), fixture, runTestKeyID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "fixture=process-success\n") || !strings.Contains(stderr, "fixture=process-success-stderr\n") {
		t.Fatalf("unexpected routed output stdout=%q stderr=%q", stdout, stderr)
	}
	if !bytes.Equal(before, mustReadRunFile(t, fixture.packagePath)) {
		t.Fatal("run mutated its package")
	}
}

func TestRunCommandPreservesChildExit23AfterOutput(t *testing.T) {
	fixture := newRunCommandFixture(t, "process_failure", true)
	stdout, stderr, err := executeRunCommand(t, context.Background(), fixture, runTestKeyID)
	if ExitCode(err) != 23 {
		t.Fatalf("exit = %d, want 23: %v", ExitCode(err), err)
	}
	if stdout != "fixture=process-failure\n" || stderr != "fixture=process-failure-stderr\n" {
		t.Fatalf("output missing before exit: stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRunCommandRejectsWrongTrustEvidence(t *testing.T) {
	fixture := newRunCommandFixture(t, "process_success", true)
	t.Run("wrong key ID", func(t *testing.T) {
		_, _, err := executeRunCommand(t, context.Background(), fixture, "other-key")
		if err == nil || ExitCode(err) != 1 {
			t.Fatalf("expected trust failure, got %v", err)
		}
	})
	t.Run("wrong public key", func(t *testing.T) {
		otherPublic, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		wrong := fixture
		wrong.keyPath = writeRunTrustedKeyFile(t, "wrong.pem", validRunTrustedKeyPEM(t, otherPublic))
		_, _, err = executeRunCommand(t, context.Background(), wrong, runTestKeyID)
		if err == nil || ExitCode(err) != 1 {
			t.Fatalf("expected signature failure, got %v", err)
		}
	})
}

func TestRunCommandRejectsMalformedKeyAndNonPackageInput(t *testing.T) {
	fixture := newRunCommandFixture(t, "process_success", true)
	fixture.keyPath = writeRunTrustedKeyFile(t, "bad.pem", []byte("not pem"))
	_, _, err := executeRunCommand(t, context.Background(), fixture, runTestKeyID)
	if !errors.Is(err, compiler.ErrInvalidTrustKey) {
		t.Fatalf("expected invalid trust key, got %v", err)
	}
	fixture.packagePath = filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(fixture.packagePath, []byte("not a package"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err = executeRunCommand(t, context.Background(), fixture, runTestKeyID)
	if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
		t.Fatalf("expected package-input rejection, got %v", err)
	}
}

func TestRunCommandRejectsTamperedPackage(t *testing.T) {
	fixture := newRunCommandFixture(t, "process_success", true)
	data := mustReadRunFile(t, fixture.packagePath)
	if len(data) < 128 {
		t.Fatal("package fixture unexpectedly small")
	}
	data[len(data)/2] ^= 0xff
	if err := os.WriteFile(fixture.packagePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := executeRunCommand(t, context.Background(), fixture, runTestKeyID)
	if err == nil || ExitCode(err) != 1 {
		t.Fatalf("expected tamper rejection, got %v", err)
	}
}

func TestRunCommandOutputTruncationPreservesSuccess(t *testing.T) {
	fixture := newRunCommandFixture(t, "process_success/output", true)
	stdout, stderr, err := executeRunCommand(t, context.Background(), fixture, runTestKeyID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stdout) != runProcessOutputLimit {
		t.Fatalf("stdout bytes = %d, want %d", len(stdout), runProcessOutputLimit)
	}
	if !strings.Contains(stderr, "forge: child stdout truncated after 1048576 bytes\n") || !strings.Contains(stderr, "forge: child stderr truncated after 1048576 bytes\n") {
		t.Fatalf("missing truncation warnings: tail=%q", stderr[len(stderr)-min(len(stderr), 200):])
	}
}

func TestRunCommandContextCancellationMapsTo130(t *testing.T) {
	fixture := newRunCommandFixture(t, "process_wait", true)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(2*time.Second, cancel)
	_, _, err := executeRunCommand(t, ctx, fixture, runTestKeyID)
	if ExitCode(err) != 130 || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation exit = %d, err=%v", ExitCode(err), err)
	}
}

func TestRunCommandRejectsUnsignedV2(t *testing.T) {
	fixture := newRunCommandFixture(t, "process_success", false)
	_, _, err := executeRunCommand(t, context.Background(), fixture, runTestKeyID)
	if err == nil || ExitCode(err) != 1 {
		t.Fatalf("expected strict signature rejection, got %v", err)
	}
}

func TestRunCommandRejectsSignedV1(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := compiler.NewEd25519Signer(runTestKeyID, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.Join(t.TempDir(), "signed-v1.zip")
	bundle := compiler.ArtifactBundle{
		ManifestName: "legacy", ManifestVersion: "v1",
		Artifacts: []compiler.Artifact{{Module: "legacy", Version: "v1", ImportPath: "example.com/legacy"}},
	}
	if err := compiler.NewZIPPackagerWithSigner(signer).Package(bundle, map[string][]byte{"legacy@v1": []byte("legacy")}, packagePath); err != nil {
		t.Fatal(err)
	}
	fixture := runCommandFixture{
		packagePath: packagePath,
		keyPath:     writeRunTrustedKeyFile(t, "public.pem", validRunTrustedKeyPEM(t, publicKey)),
	}
	_, _, err = executeRunCommand(t, context.Background(), fixture, runTestKeyID)
	if err == nil || ExitCode(err) != 1 {
		t.Fatalf("expected signed v1 rejection, got %v", err)
	}
}

func TestForgeRunHelperProcessExit23(t *testing.T) {
	if os.Getenv("FORGE_RUN_HELPER") == "1" {
		root := NewRootCommand()
		separator := 0
		for index, argument := range os.Args {
			if argument == "--" {
				separator = index
				break
			}
		}
		root.SetArgs(os.Args[separator+1:])
		os.Exit(ExitCode(root.Execute()))
	}
	fixture := newRunCommandFixture(t, "process_failure", true)
	child := exec.Command(os.Args[0], "-test.run=TestForgeRunHelperProcessExit23", "--", "run", fixture.packagePath, "--trusted-key", fixture.keyPath, "--key-id", runTestKeyID)
	child.Env = append(os.Environ(), "FORGE_RUN_HELPER=1")
	output, err := child.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 23 {
		t.Fatalf("OS exit err=%v output=%q", err, output)
	}
}

func executeRunCommand(t *testing.T, ctx context.Context, fixture runCommandFixture, keyID string) (string, string, error) {
	t.Helper()
	cmd := NewRootCommandWithApplication(bootstrap.NewApplication())
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"run", fixture.packagePath, "--trusted-key", fixture.keyPath, "--key-id", keyID})
	ctx = context.WithValue(ctx, applicationContextKey{}, bootstrap.NewApplication())
	err := cmd.ExecuteContext(ctx)
	return stdout.String(), stderr.String(), err
}

func newRunCommandFixture(t *testing.T, fixture string, signed bool) runCommandFixture {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	packager := compiler.NewZIPPackager()
	if signed {
		signer, err := compiler.NewEd25519Signer(runTestKeyID, privateKey)
		if err != nil {
			t.Fatal(err)
		}
		packager = compiler.NewZIPPackagerWithSigner(signer)
	}
	sources := compiler.NewPackageSourceRegistry()
	module := strings.ReplaceAll(fixture, "/", "-")
	if err := sources.Register(compiler.PackageSource{Name: module, Version: "v1", ImportPath: "github.com/kaizenforyou91/forge/runtime/testdata/" + fixture}); err != nil {
		t.Fatal(err)
	}
	builder, err := compiler.NewGoApplicationExecutableBuilder(compiler.NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	coordinator, err := compiler.NewRunnablePackageCompiler(sources, builder, packager)
	if err != nil {
		t.Fatal(err)
	}
	packagePath := filepath.Join(t.TempDir(), module+".zip")
	root := runRepositoryRoot(t)
	err = coordinator.Compile(context.Background(), compiler.RunnablePackageRequest{
		Plan:       manifest.BuildPlan{ManifestName: "run-test", ManifestVersion: "v1", Steps: []manifest.BuildStep{{Module: module + "@v1"}}},
		Entrypoint: compiler.RuntimeEntrypoint{Module: module, Version: "v1"}, WorkingDirectory: root, OutputPath: packagePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	keyPath := writeRunTrustedKeyFile(t, "public.pem", validRunTrustedKeyPEM(t, publicKey))
	return runCommandFixture{packagePath: packagePath, keyPath: keyPath, publicKey: publicKey}
}

func runRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repository root")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func mustReadRunFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
