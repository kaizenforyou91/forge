package runtime

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/kaizenforyou91/forge/pkg/manifest"
)

func TestProcessRunnerExecutionIntegration(t *testing.T) {
	repositoryDirectory := repositoryRoot(t)
	fixtureDirectory := filepath.Join(repositoryDirectory, "runtime", "testdata", "process_success")
	fixtureSourcePath := filepath.Join(fixtureDirectory, "main.go")
	fixtureSourceBefore, err := os.ReadFile(fixtureSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	fixtureFilesBefore := directoryFiles(t, fixtureDirectory)

	const (
		module     = "process-success"
		version    = "v1"
		importPath = processRunnerFixtureImportBase + "process_success"
	)
	entrypoint := compiler.RuntimeEntrypoint{Module: module, Version: version}
	signer, trustStore := newTestSignerAndTrustStore(t)
	sources := compiler.NewPackageSourceRegistry()
	if err := sources.Register(compiler.PackageSource{
		Name:       module,
		Version:    version,
		ImportPath: importPath,
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

	packagePath := filepath.Join(t.TempDir(), "process-success.zip")
	if err := coordinator.Compile(context.Background(), compiler.RunnablePackageRequest{
		Plan: manifest.BuildPlan{
			ManifestName:    "process-runner-integration",
			ManifestVersion: "v1",
			Steps: []manifest.BuildStep{{
				Module: module + "@" + version,
			}},
		},
		Entrypoint:       entrypoint,
		WorkingDirectory: repositoryDirectory,
		OutputPath:       packagePath,
	}); err != nil {
		t.Fatal(err)
	}
	packageBefore, err := os.ReadFile(packagePath)
	if err != nil {
		t.Fatal(err)
	}

	loader, err := NewVerifiedRunnablePackageLoader(trustStore)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := loader.Load(packagePath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SignerKeyID() != testSignerKeyID || loaded.Entrypoint() != entrypoint {
		t.Fatalf(
			"strict load evidence = signer %q entrypoint %#v",
			loaded.SignerKeyID(),
			loaded.Entrypoint(),
		)
	}
	if loaded.PackageFormatVersion() != 2 || loaded.BundleSchemaVersion() != 2 {
		t.Fatalf(
			"strict loaded version pair = (%d,%d), want (2,2)",
			loaded.PackageFormatVersion(),
			loaded.BundleSchemaVersion(),
		)
	}

	first := materializeAndRunIntegrationExecutable(t, loaded, entrypoint)
	if packageAfter, err := os.ReadFile(packagePath); err != nil {
		t.Fatal(err)
	} else if !bytes.Equal(packageAfter, packageBefore) {
		t.Fatal("normal load/materialize/execute lifecycle mutated the signed package")
	}
	firstDirectory := first.directory
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(firstDirectory); !os.IsNotExist(err) {
		t.Fatalf("first execution directory remains after Close: %v", err)
	}

	// Replace the ZIP only after strict loading. A second materialization and
	// execution from the detached verified result must not reopen the package.
	if err := os.WriteFile(packagePath, []byte("package replaced after strict load"), 0o600); err != nil {
		t.Fatal(err)
	}
	second := materializeAndRunIntegrationExecutable(t, loaded, entrypoint)
	secondDirectory := second.directory
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(secondDirectory); !os.IsNotExist(err) {
		t.Fatalf("detached execution directory remains after Close: %v", err)
	}

	fixtureSourceAfter, err := os.ReadFile(fixtureSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fixtureSourceAfter, fixtureSourceBefore) {
		t.Fatal("process fixture source changed during trusted execution integration")
	}
	if fixtureFilesAfter := directoryFiles(t, fixtureDirectory); !equalStrings(fixtureFilesBefore, fixtureFilesAfter) {
		t.Fatalf(
			"process fixture directory changed: before=%v after=%v",
			fixtureFilesBefore,
			fixtureFilesAfter,
		)
	}
}

func materializeAndRunIntegrationExecutable(
	t *testing.T,
	loaded VerifiedRunnablePackage,
	entrypoint compiler.RuntimeEntrypoint,
) *MaterializedExecutable {
	t.Helper()

	materialized, err := NewSecureExecutableMaterializer().Materialize(loaded)
	if err != nil {
		t.Fatal(err)
	}
	directory := materialized.directory
	t.Cleanup(func() { _ = materialized.Close() })

	process, err := NewProcessRunner().Start(context.Background(), materialized)
	if err != nil {
		t.Fatal(err)
	}
	if process.Entrypoint() != entrypoint || process.SignerKeyID() != testSignerKeyID {
		t.Fatalf(
			"running trust evidence = signer %q entrypoint %#v",
			process.SignerKeyID(),
			process.Entrypoint(),
		)
	}
	result, err := process.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 || result.Canceled || result.Terminated {
		t.Fatalf("real execution result = %#v", result)
	}
	if outputValue(result.Stdout, "fixture") != "process-success" ||
		outputValue(result.Stderr, "fixture") != "process-success-stderr" {
		t.Fatalf("real execution output: stdout=%q stderr=%q", result.Stdout, result.Stderr)
	}
	if _, err := os.Lstat(directory); err != nil {
		t.Fatalf("process exit unexpectedly auto-cleaned materialization: %v", err)
	}
	return materialized
}
