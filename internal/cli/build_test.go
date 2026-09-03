package cli

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/internal/bootstrap"
	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/kaizenforyou91/forge/pkg/compiler"
	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
	"github.com/kaizenforyou91/forge/pkg/registry"
	"github.com/spf13/cobra"
)

type buildCommandRegistrySnapshot struct {
	packages []registry.Package
	sources  []compiler.PackageSource
}

func buildCommandRegistries(
	t *testing.T,
	application *app.App,
) (*registry.Registry, *compiler.PackageSourceRegistry) {
	t.Helper()

	packages, err := resolveRegistry(application)
	if err != nil {
		t.Fatal(err)
	}

	sources, err := resolveSourceRegistry(application)
	if err != nil {
		t.Fatal(err)
	}

	return packages, sources
}

func snapshotBuildCommandRegistries(
	packages *registry.Registry,
	sources *compiler.PackageSourceRegistry,
) buildCommandRegistrySnapshot {
	return buildCommandRegistrySnapshot{
		packages: packages.List(),
		sources:  sources.List(),
	}
}

func requireBuildCommandRegistrySnapshot(
	t *testing.T,
	packages *registry.Registry,
	sources *compiler.PackageSourceRegistry,
	want buildCommandRegistrySnapshot,
) {
	t.Helper()

	got := snapshotBuildCommandRegistries(packages, sources)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected registry state %#v, got %#v", want, got)
	}
}

func seedBuildCommandRegistries(
	t *testing.T,
	packages *registry.Registry,
	sources *compiler.PackageSourceRegistry,
) {
	t.Helper()

	if err := packages.EnsureAll([]registry.Package{
		{Name: "existing", Version: "v1"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := sources.EnsureAll([]compiler.PackageSource{
		{
			Name:       "existing",
			Version:    "v1",
			ImportPath: "example.com/forge/existing",
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func writeBuildCommandManifest(
	t *testing.T,
	directory,
	name,
	data string,
) string {
	t.Helper()

	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	return path
}

func executeBuildCommand(
	application *app.App,
	manifestPath,
	outputPath string,
) error {
	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	return cmd.Execute()
}

func requireBuildCommandOutputAbsent(
	t *testing.T,
	outputPath string,
) {
	t.Helper()

	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Fatalf("expected no output at %q, got %v", outputPath, err)
	}
}

func readBuildCommandBundle(
	t *testing.T,
	outputPath string,
) compiler.ArtifactBundle {
	t.Helper()

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != "bundle.json" {
			continue
		}

		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}

		data, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}

		bundle, err := compiler.UnmarshalArtifactBundle(data)
		if err != nil {
			t.Fatal(err)
		}

		return bundle
	}

	t.Fatal("bundle.json not found")
	return compiler.ArtifactBundle{}
}

func requireBuildCommandAdmissionFailureDoesNotMutateRegistries(
	t *testing.T,
	manifestData string,
	checkError func(*testing.T, error),
) {
	t.Helper()

	directory := t.TempDir()
	manifestPath := writeBuildCommandManifest(
		t,
		directory,
		"forge.yaml",
		manifestData,
	)
	outputPath := filepath.Join(directory, "output.zip")
	application := bootstrap.NewApplication()
	packages, sources := buildCommandRegistries(t, application)
	seedBuildCommandRegistries(t, packages, sources)
	before := snapshotBuildCommandRegistries(packages, sources)

	err := executeBuildCommand(application, manifestPath, outputPath)
	if err == nil {
		t.Fatal("expected build failure")
	}

	checkError(t, err)
	requireBuildCommandRegistrySnapshot(t, packages, sources, before)
	requireBuildCommandOutputAbsent(t, outputPath)
}

func TestBuildCommandCreatesPackage(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "build", "demo-v1.zip")

	manifestData := `
version: v1
name: demo
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)

	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf(
			"expected package %q to exist: %v",
			outputPath,
			err,
		)
	}

	if !strings.Contains(
		stdout.String(),
		"Build completed:",
	) {
		t.Fatalf(
			"expected build completion output, got %q",
			stdout.String(),
		)
	}
}

func TestBuildCommandOutputRemainsPackageFormatV1IdentityPayload(t *testing.T) {
	dir := t.TempDir()
	manifestPath := writeBuildCommandManifest(t, dir, "forge.yaml", `
version: v1
name: demo
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge
`)
	outputPath := filepath.Join(dir, "demo-v1.zip")
	if err := executeBuildCommand(bootstrap.NewApplication(), manifestPath, outputPath); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entries := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		entries[file.Name] = data
	}

	wantMetadata := []byte(`{"package_format_version":1,"bundle_schema_version":1}`)
	if !bytes.Equal(entries["package.json"], wantMetadata) {
		t.Fatalf("expected %s, got %s", wantMetadata, entries["package.json"])
	}
	wantIdentityPayload := []byte("forge@v1")
	if !bytes.Equal(entries["artifacts/forge/v1/artifact"], wantIdentityPayload) {
		t.Fatalf(
			"expected identity payload %q, got %q",
			wantIdentityPayload,
			entries["artifacts/forge/v1/artifact"],
		)
	}
}

func TestBuildCommandOutputRemainsPackageFormatV1WithEntrypoint(t *testing.T) {
	directory := t.TempDir()
	manifestPath := writeBuildCommandManifest(t, directory, "forge.yaml", `
version: v1
name: demo
entrypoint:
  module: forge
  version: v1
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge
`)
	outputPath := filepath.Join(directory, "demo-v1-with-entrypoint.zip")
	if err := executeBuildCommand(bootstrap.NewApplication(), manifestPath, outputPath); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entries := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		entries[file.Name] = data
	}

	wantMetadata := []byte(`{"package_format_version":1,"bundle_schema_version":1}`)
	if !bytes.Equal(entries["package.json"], wantMetadata) {
		t.Fatalf("expected %s, got %s", wantMetadata, entries["package.json"])
	}
	wantIdentityPayload := []byte("forge@v1")
	if !bytes.Equal(entries["artifacts/forge/v1/artifact"], wantIdentityPayload) {
		t.Fatalf(
			"expected identity payload %q, got %q",
			wantIdentityPayload,
			entries["artifacts/forge/v1/artifact"],
		)
	}
}

func TestBuildCommandCreatesDeterministicPackageEntries(
	t *testing.T,
) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "demo.zip")

	manifestData := `
version: v1
name: demo
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	expected := map[string]bool{
		"bundle.json":                 false,
		"integrity.json":              false,
		"artifacts/forge/v1/artifact": false,
	}

	for _, file := range reader.File {
		if _, ok := expected[file.Name]; ok {
			expected[file.Name] = true
		}
	}

	for path, found := range expected {
		if !found {
			t.Fatalf(
				"expected ZIP entry %q",
				path,
			)
		}
	}
}

func TestBuildCommandCanExecuteTwiceWithSameApplication(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	firstOutputPath := filepath.Join(dir, "first.zip")
	secondOutputPath := filepath.Join(dir, "second.zip")

	manifestData := `
version: v1
name: demo
modules:
  - name: compiler
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	for _, outputPath := range []string{
		firstOutputPath,
		secondOutputPath,
	} {
		cmd := NewRootCommandWithApplication(application)
		cmd.SetArgs([]string{
			"build",
			manifestPath,
			"--output",
			outputPath,
		})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}

	for _, outputPath := range []string{
		firstOutputPath,
		secondOutputPath,
	} {
		if _, err := os.Stat(outputPath); err != nil {
			t.Fatalf(
				"expected package %q to exist: %v",
				outputPath,
				err,
			)
		}
	}
}

func TestBuildCommandRepeatedExecutionProducesDeterministicPackage(
	t *testing.T,
) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	firstOutputPath := filepath.Join(dir, "first.zip")
	secondOutputPath := filepath.Join(dir, "second.zip")

	manifestData := `
version: v1
name: demo
modules:
  - name: compiler
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	for _, outputPath := range []string{
		firstOutputPath,
		secondOutputPath,
	} {
		cmd := NewRootCommandWithApplication(application)
		cmd.SetArgs([]string{
			"build",
			manifestPath,
			"--output",
			outputPath,
		})

		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}

	first, err := os.ReadFile(firstOutputPath)
	if err != nil {
		t.Fatal(err)
	}

	second, err := os.ReadFile(secondOutputPath)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatal("expected repeated builds to produce identical packages")
	}
}

func TestBuildCommandRejectsConflictingImportPathWithSameApplication(
	t *testing.T,
) {
	dir := t.TempDir()

	firstManifestPath := filepath.Join(dir, "first.yaml")
	secondManifestPath := filepath.Join(dir, "second.yaml")
	firstOutputPath := filepath.Join(dir, "first.zip")
	secondOutputPath := filepath.Join(dir, "second.zip")

	firstManifestData := `
version: v1
name: demo
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge
`

	secondManifestData := `
version: v1
name: demo
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
`

	if err := os.WriteFile(
		firstManifestPath,
		[]byte(firstManifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		secondManifestPath,
		[]byte(secondManifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	firstCommand := NewRootCommandWithApplication(application)
	firstCommand.SetArgs([]string{
		"build",
		firstManifestPath,
		"--output",
		firstOutputPath,
	})

	if err := firstCommand.Execute(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(firstOutputPath); err != nil {
		t.Fatalf(
			"expected package %q to exist: %v",
			firstOutputPath,
			err,
		)
	}

	secondCommand := NewRootCommandWithApplication(application)
	secondCommand.SetArgs([]string{
		"build",
		secondManifestPath,
		"--output",
		secondOutputPath,
	})

	err := secondCommand.Execute()

	if !errors.Is(err, compiler.ErrPackageSourceConflict) {
		t.Fatalf(
			"expected ErrPackageSourceConflict, got %v",
			err,
		)
	}

	if _, statErr := os.Stat(secondOutputPath); !os.IsNotExist(statErr) {
		t.Fatalf(
			"conflicting package must not be created, stat error: %v",
			statErr,
		)
	}
}

func TestBuildCommandSupportsJSONManifest(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.json")
	outputPath := filepath.Join(dir, "demo.json.zip")

	manifestData := `{
	"version": "v1",
	"name": "demo",
	"modules": [
		{
			"name": "forge",
			"version": "v1",
			"import_path": "github.com/kaizenforyou91/forge/cmd/forge"
		}
	]
}`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf(
			"expected JSON build package %q: %v",
			outputPath,
			err,
		)
	}
}

func TestBuildCommandUsesDefaultOutputPath(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")

	manifestData := `
version: v1
name: demo
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{
		"build",
		manifestPath,
	})

	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	expectedPath := filepath.Join(
		"build",
		"demo-v1.zip",
	)

	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf(
			"expected default output package %q: %v",
			expectedPath,
			err,
		)
	}

	t.Cleanup(func() {
		_ = os.Remove(expectedPath)
	})

	if !strings.Contains(
		stdout.String(),
		"Build completed:",
	) {
		t.Fatalf(
			"expected build completion output, got %q",
			stdout.String(),
		)
	}
}

func TestBuildCommandRequiresManifestArgument(t *testing.T) {
	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{"build"})

	err := cmd.Execute()

	if err == nil {
		t.Fatal("expected missing manifest argument error")
	}
}

func TestBuildCommandRejectsUnsupportedManifestFormat(
	t *testing.T,
) {
	dir := t.TempDir()

	manifestPath := filepath.Join(
		dir,
		"forge.toml",
	)

	if err := os.WriteFile(
		manifestPath,
		[]byte("invalid = true"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{
		"build",
		manifestPath,
	})

	err := cmd.Execute()

	if err == nil {
		t.Fatal("expected unsupported manifest format error")
	}

	if !strings.Contains(
		err.Error(),
		"unsupported manifest format",
	) {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}
}

func TestLoadManifestFileYAML(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "forge.yaml")

	data := `
version: v1
name: demo
modules:
  - name: logger
    version: v1
`

	if err := os.WriteFile(
		path,
		[]byte(data),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	got, err := loadManifestFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "demo" {
		t.Fatalf(
			"expected manifest name %q, got %q",
			"demo",
			got.Name,
		)
	}

	if got.Version != "v1" {
		t.Fatalf(
			"expected manifest version %q, got %q",
			"v1",
			got.Version,
		)
	}

	if len(got.Modules) != 1 {
		t.Fatalf(
			"expected 1 module, got %d",
			len(got.Modules),
		)
	}
}

func TestLoadManifestFileJSON(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "forge.json")

	data := `{
	"version": "v1",
	"name": "demo",
	"modules": [
		{
			"name": "logger",
			"version": "v1"
		}
	]
}`

	if err := os.WriteFile(
		path,
		[]byte(data),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	got, err := loadManifestFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "demo" {
		t.Fatalf(
			"expected manifest name %q, got %q",
			"demo",
			got.Name,
		)
	}
}

func TestResolveRegistryRejectsNilApplication(t *testing.T) {
	_, err := resolveRegistry(nil)

	if err == nil {
		t.Fatal("expected nil application error")
	}
}

func TestResolveEngineRejectsNilApplication(t *testing.T) {
	_, err := resolveEngine(nil)

	if err == nil {
		t.Fatal("expected nil application error")
	}
}

func TestBuildCommandContextContainsApplication(
	t *testing.T,
) {
	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)

	got, err := ApplicationFromContext(cmd.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got != application {
		t.Fatal("expected injected application")
	}
}

func TestBuildCommandIsRegistered(t *testing.T) {
	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)

	buildCommand, _, err := cmd.Find([]string{"build"})
	if err != nil {
		t.Fatal(err)
	}

	if buildCommand == nil {
		t.Fatal("expected build command")
	}

	if buildCommand.Name() != "build" {
		t.Fatalf(
			"expected command name %q, got %q",
			"build",
			buildCommand.Name(),
		)
	}
}

func TestBuildCommandRejectsInvalidManifestContent(
	t *testing.T,
) {
	dir := t.TempDir()

	path := filepath.Join(dir, "forge.yaml")

	if err := os.WriteFile(
		path,
		[]byte("version: [invalid"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	_, err := loadManifestFile(path)

	if err == nil {
		t.Fatal("expected manifest parsing error")
	}
}

func TestBuildCommandRejectsStrictManifestViolationsBeforeAdmission(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		data     []byte
	}{
		{
			name:     "duplicate JSON key",
			filename: "forge.json",
			data:     []byte(`{"version":"v1","version":"v2","name":"demo","modules":[]}`),
		},
		{
			name:     "unknown JSON field",
			filename: "forge.json",
			data:     []byte(`{"version":"v1","name":"demo","modules":[],"extra":true}`),
		},
		{
			name:     "malformed UTF-8 JSON",
			filename: "forge.json",
			data:     append([]byte(`{"version":"v1","name":"`), append([]byte{0xff}, []byte(`","modules":[]}`)...)...),
		},
		{
			name:     "multiple YAML documents",
			filename: "forge.yaml",
			data:     []byte("version: v1\nname: demo\nmodules: []\n---\n{}\n"),
		},
		{
			name:     "YAML anchor",
			filename: "forge.yaml",
			data:     []byte("version: &version v1\nname: demo\nmodules: []\n"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			manifestPath := filepath.Join(directory, test.filename)
			if err := os.WriteFile(manifestPath, test.data, 0o644); err != nil {
				t.Fatal(err)
			}
			outputPath := filepath.Join(directory, "must-not-exist.zip")
			application := bootstrap.NewApplication()
			packages, sources := buildCommandRegistries(t, application)
			before := snapshotBuildCommandRegistries(packages, sources)

			err := executeBuildCommand(application, manifestPath, outputPath)
			requireInvalidManifestCode(t, err)
			requireBuildCommandRegistrySnapshot(t, packages, sources, before)
			requireBuildCommandOutputAbsent(t, outputPath)
		})
	}
}

func TestBuildCommandOutputErrorIsPropagated(
	t *testing.T,
) {
	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)

	// This exercises the command's normal validation layer without
	// requiring a real package output.
	cmd.SetArgs([]string{
		"build",
		"missing.yaml",
	})

	err := cmd.Execute()

	if err == nil {
		t.Fatal("expected missing manifest error")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf(
			"expected os.ErrNotExist, got %v",
			err,
		)
	}
}

func requireInvalidManifestCode(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected invalid manifest error")
	}
	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if forgeErr.Code != forgeerrors.CodeInvalidManifest {
		t.Fatalf("expected code %s, got %s", forgeerrors.CodeInvalidManifest, forgeErr.Code)
	}
}

func TestBuildCommandOutputCanBeReadBack(
	t *testing.T,
) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "demo.zip")

	manifestData := `
version: v1
name: demo
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	foundBundle := false
	foundIntegrity := false

	for _, file := range reader.File {
		switch file.Name {
		case "bundle.json":
			foundBundle = true
		case "integrity.json":
			foundIntegrity = true
		}
	}

	if !foundBundle {
		t.Fatal("expected bundle.json")
	}

	if !foundIntegrity {
		t.Fatal("expected integrity.json")
	}
}

func TestBuildCommandDefaultOutputUsesManifestIdentity(
	t *testing.T,
) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")

	data := `
version: v2
name: sample
modules:
  - name: sample
    version: v2
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(data),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	got, err := loadManifestFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(
		"build",
		got.Name+"-"+got.Version+".zip",
	)

	expected := filepath.Join(
		"build",
		"sample-v2.zip",
	)

	if output != expected {
		t.Fatalf(
			"expected %q, got %q",
			expected,
			output,
		)
	}
}

func TestBuildCommandDoesNotRequireExplicitOutput(
	t *testing.T,
) {
	var command *cobra.Command

	application := bootstrap.NewApplication()
	root := NewRootCommandWithApplication(application)

	command, _, err := root.Find([]string{"build"})
	if err != nil {
		t.Fatal(err)
	}

	if command == nil {
		t.Fatal("expected build command")
	}

	flag := command.Flags().Lookup("output")
	if flag == nil {
		t.Fatal("expected output flag")
	}
}

func TestBuildCommandRejectsMissingDependency(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "missing-dependency.zip")

	manifestData := `
version: v1
name: missing-dependency
modules:
  - name: compiler
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
    dependencies:
      - name: core
        version: v1
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommandWithApplication(
		bootstrap.NewApplication(),
	)

	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing dependency error")
	}

	if !strings.Contains(
		err.Error(),
		"dependency",
	) {
		t.Fatalf(
			"expected dependency error, got %v",
			err,
		)
	}

	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf(
			"package must not be created, stat error: %v",
			statErr,
		)
	}
}

func TestBuildCommandRejectsCircularDependency(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "cycle.zip")

	manifestData := `
version: v1
name: cycle
modules:
  - name: compiler
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
    dependencies:
      - name: core
        version: v1

  - name: core
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/app
    dependencies:
      - name: compiler
        version: v1
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommandWithApplication(
		bootstrap.NewApplication(),
	)

	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected circular dependency error")
	}

	if !strings.Contains(
		err.Error(),
		"circular",
	) {
		t.Fatalf(
			"expected circular dependency error, got %v",
			err,
		)
	}

	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf(
			"package must not be created, stat error: %v",
			statErr,
		)
	}
}

func TestBuildCommandRejectsDuplicateModule(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "duplicate.zip")

	manifestData := `
version: v1
name: duplicate
modules:
  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge

  - name: forge
    version: v1
    import_path: github.com/kaizenforyou91/forge/cmd/forge
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommandWithApplication(
		bootstrap.NewApplication(),
	)

	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected duplicate module error")
	}

	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf(
			"package must not be created, stat error: %v",
			statErr,
		)
	}
}

func TestBuildCommandRejectsMissingImportPath(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "missing-source.zip")

	manifestData := `
version: v1
name: missing-source
modules:
  - name: forge
    version: v1
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommandWithApplication(
		bootstrap.NewApplication(),
	)

	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing import_path error")
	}

	if !errors.Is(err, compiler.ErrInvalidPackageSource) {
		t.Fatalf(
			"expected ErrInvalidPackageSource, got %v",
			err,
		)
	}

	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf(
			"package must not be created, stat error: %v",
			statErr,
		)
	}
}

func TestBuildCommandSupportsMultipleModulesWithDependencies(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "multi.zip")

	manifestData := `
version: v1
name: multi-demo
modules:
  - name: compiler
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
    dependencies:
      - name: core
        version: v1

  - name: core
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/app
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	var stdout strings.Builder
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf(
			"expected package %q to exist: %v",
			outputPath,
			err,
		)
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	expected := map[string]bool{
		"bundle.json":                    false,
		"integrity.json":                 false,
		"artifacts/core/v1/artifact":     false,
		"artifacts/compiler/v1/artifact": false,
	}

	for _, file := range reader.File {
		if _, ok := expected[file.Name]; ok {
			expected[file.Name] = true
		}
	}

	for path, found := range expected {
		if !found {
			t.Fatalf("expected ZIP entry %q", path)
		}
	}

	if !strings.Contains(stdout.String(), "Build completed:") {
		t.Fatalf(
			"expected build completion output, got %q",
			stdout.String(),
		)
	}
}

func TestBuildCommandPreservesDependencyFirstArtifactOrder(t *testing.T) {
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, "forge.yaml")
	outputPath := filepath.Join(dir, "ordered.zip")

	manifestData := `
version: v1
name: ordered-demo
modules:
  - name: compiler
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
    dependencies:
      - name: core
        version: v1

  - name: core
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/app
`

	if err := os.WriteFile(
		manifestPath,
		[]byte(manifestData),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()

	cmd := NewRootCommandWithApplication(application)
	cmd.SetArgs([]string{
		"build",
		manifestPath,
		"--output",
		outputPath,
	})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.Name != "bundle.json" {
			continue
		}

		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()

		data, err := io.ReadAll(stream)
		if err != nil {
			t.Fatal(err)
		}

		bundle, err := compiler.UnmarshalArtifactBundle(data)
		if err != nil {
			t.Fatal(err)
		}

		if len(bundle.Artifacts) != 2 {
			t.Fatalf(
				"expected 2 artifacts, got %d",
				len(bundle.Artifacts),
			)
		}

		if bundle.Artifacts[0].Module != "core" ||
			bundle.Artifacts[0].Version != "v1" {
			t.Fatalf(
				"expected core@v1 first, got %#v",
				bundle.Artifacts[0],
			)
		}

		if bundle.Artifacts[1].Module != "compiler" ||
			bundle.Artifacts[1].Version != "v1" {
			t.Fatalf(
				"expected compiler@v1 second, got %#v",
				bundle.Artifacts[1],
			)
		}

		return
	}

	t.Fatal("bundle.json not found")
}

func TestBuildCommandInvalidManifestDoesNotMutateRegistries(
	t *testing.T,
) {
	requireBuildCommandAdmissionFailureDoesNotMutateRegistries(
		t,
		`version: v1
modules:
  - name: app
    version: v1
    import_path: example.com/forge/app
`,
		func(t *testing.T, err error) {
			t.Helper()
			if !strings.Contains(err.Error(), "manifest.name") {
				t.Fatalf("expected manifest validation error, got %v", err)
			}
		},
	)
}

func TestBuildCommandMissingImportPathDoesNotMutateRegistries(
	t *testing.T,
) {
	requireBuildCommandAdmissionFailureDoesNotMutateRegistries(
		t,
		`version: v1
name: missing-source
modules:
  - name: app
    version: v1
    import_path: "   "
`,
		func(t *testing.T, err error) {
			t.Helper()
			if !errors.Is(err, compiler.ErrInvalidPackageSource) {
				t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
			}
		},
	)
}

func TestBuildCommandMissingDependencyDoesNotMutateRegistries(
	t *testing.T,
) {
	requireBuildCommandAdmissionFailureDoesNotMutateRegistries(
		t,
		`version: v1
name: missing-dependency
modules:
  - name: app
    version: v1
    import_path: example.com/forge/app
    dependencies:
      - name: missing
        version: v1
`,
		func(t *testing.T, err error) {
			t.Helper()
			if !strings.Contains(err.Error(), "dependency") {
				t.Fatalf("expected dependency error, got %v", err)
			}
		},
	)
}

func TestBuildCommandCycleDoesNotMutateRegistries(t *testing.T) {
	requireBuildCommandAdmissionFailureDoesNotMutateRegistries(
		t,
		`version: v1
name: cycle
modules:
  - name: a
    version: v1
    import_path: example.com/forge/a
    dependencies:
      - name: b
        version: v1
  - name: b
    version: v1
    import_path: example.com/forge/b
    dependencies:
      - name: a
        version: v1
`,
		func(t *testing.T, err error) {
			t.Helper()
			if !strings.Contains(err.Error(), "circular") {
				t.Fatalf("expected circular dependency error, got %v", err)
			}
		},
	)
}

func TestBuildCommandSourceConflictDoesNotPartiallyMutateRegistries(
	t *testing.T,
) {
	directory := t.TempDir()
	manifestPath := writeBuildCommandManifest(
		t,
		directory,
		"forge.yaml",
		`version: v1
name: conflict
modules:
  - name: a
    version: v1
    import_path: example.com/forge/a
  - name: b
    version: v1
    import_path: example.com/forge/requested-b
  - name: c
    version: v1
    import_path: example.com/forge/c
`,
	)
	outputPath := filepath.Join(directory, "conflict.zip")
	application := bootstrap.NewApplication()
	packages, sources := buildCommandRegistries(t, application)
	canonicalPackage := registry.Package{Name: "b", Version: "v1"}
	canonicalSource := compiler.PackageSource{
		Name:       "b",
		Version:    "v1",
		ImportPath: "example.com/forge/canonical-b",
	}

	if err := packages.EnsureAll([]registry.Package{canonicalPackage}); err != nil {
		t.Fatal(err)
	}
	if err := sources.EnsureAll([]compiler.PackageSource{canonicalSource}); err != nil {
		t.Fatal(err)
	}

	before := snapshotBuildCommandRegistries(packages, sources)
	err := executeBuildCommand(application, manifestPath, outputPath)
	if !errors.Is(err, compiler.ErrPackageSourceConflict) {
		t.Fatalf("expected ErrPackageSourceConflict, got %v", err)
	}

	requireBuildCommandRegistrySnapshot(t, packages, sources, before)
	requireBuildCommandOutputAbsent(t, outputPath)

	for _, name := range []string{"a", "c"} {
		if _, getErr := packages.Get(name, "v1"); !errors.Is(
			getErr,
			registry.ErrPackageNotFound,
		) {
			t.Fatalf("expected package %s@v1 to be absent, got %v", name, getErr)
		}

		if _, resolveErr := sources.Resolve(name, "v1"); !errors.Is(
			resolveErr,
			compiler.ErrPackageSourceNotFound,
		) {
			t.Fatalf("expected source %s@v1 to be absent, got %v", name, resolveErr)
		}
	}

	got, resolveErr := sources.Resolve("b", "v1")
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	if !reflect.DeepEqual(got, canonicalSource) {
		t.Fatalf("expected canonical source %#v, got %#v", canonicalSource, got)
	}
}

func TestBuildCommandSuccessfulAdmissionCommitsRegistries(t *testing.T) {
	directory := t.TempDir()
	manifestPath := writeBuildCommandManifest(
		t,
		directory,
		"forge.yaml",
		`version: v1
name: admitted
modules:
  - name: compiler
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
    dependencies:
      - name: core
        version: v1
  - name: core
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/app
`,
	)
	outputPath := filepath.Join(directory, "admitted.zip")
	application := bootstrap.NewApplication()
	packages, sources := buildCommandRegistries(t, application)

	if err := executeBuildCommand(application, manifestPath, outputPath); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output package: %v", err)
	}

	wantPackages := []registry.Package{
		{Name: "compiler", Version: "v1"},
		{Name: "core", Version: "v1"},
	}
	wantSources := []compiler.PackageSource{
		{
			Name:       "compiler",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
		},
		{
			Name:       "core",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/app",
		},
	}
	requireBuildCommandRegistrySnapshot(
		t,
		packages,
		sources,
		buildCommandRegistrySnapshot{
			packages: wantPackages,
			sources:  wantSources,
		},
	)

	bundle := readBuildCommandBundle(t, outputPath)
	wantArtifactOrder := []string{"core@v1", "compiler@v1"}
	if len(bundle.Artifacts) != len(wantArtifactOrder) {
		t.Fatalf(
			"expected %d artifacts, got %d",
			len(wantArtifactOrder),
			len(bundle.Artifacts),
		)
	}

	for i, want := range wantArtifactOrder {
		got := bundle.Artifacts[i].Module + "@" + bundle.Artifacts[i].Version
		if got != want {
			t.Fatalf("artifact %d: expected %q, got %q", i, want, got)
		}
	}
}

func TestBuildCommandExecutorFailureKeepsAdmittedRegistration(
	t *testing.T,
) {
	directory := t.TempDir()
	missingImportPath := "./__forge_executor_failure_source_does_not_exist__"
	if _, err := os.Stat(missingImportPath); !os.IsNotExist(err) {
		t.Fatalf("executor failure fixture unexpectedly exists: %v", err)
	}

	manifestPath := writeBuildCommandManifest(
		t,
		directory,
		"forge.yaml",
		fmt.Sprintf(`version: v1
name: executor-failure
modules:
  - name: broken
    version: v1
    import_path: %s
`, missingImportPath),
	)
	application := bootstrap.NewApplication()
	packages, sources := buildCommandRegistries(t, application)
	firstOutput := filepath.Join(directory, "first.zip")

	err := executeBuildCommand(application, manifestPath, firstOutput)
	if !errors.Is(err, compiler.ErrCommandFailed) {
		t.Fatalf("expected ErrCommandFailed, got %v", err)
	}
	requireBuildCommandOutputAbsent(t, firstOutput)

	wantState := buildCommandRegistrySnapshot{
		packages: []registry.Package{{Name: "broken", Version: "v1"}},
		sources: []compiler.PackageSource{
			{
				Name:       "broken",
				Version:    "v1",
				ImportPath: missingImportPath,
			},
		},
	}
	requireBuildCommandRegistrySnapshot(t, packages, sources, wantState)

	secondOutput := filepath.Join(directory, "second.zip")
	err = executeBuildCommand(application, manifestPath, secondOutput)
	if !errors.Is(err, compiler.ErrCommandFailed) {
		t.Fatalf("expected retry ErrCommandFailed, got %v", err)
	}
	requireBuildCommandOutputAbsent(t, secondOutput)
	requireBuildCommandRegistrySnapshot(t, packages, sources, wantState)
}

func TestBuildCommandPackagingFailureKeepsAdmittedRegistration(
	t *testing.T,
) {
	directory := t.TempDir()
	manifestPath := writeBuildCommandManifest(
		t,
		directory,
		"forge.yaml",
		`version: v1
name: packaging-failure
modules:
  - name: compiler
    version: v1
    import_path: github.com/kaizenforyou91/forge/pkg/compiler
`,
	)
	blockedOutput := filepath.Join(directory, "blocked-output")
	if err := os.Mkdir(blockedOutput, 0o755); err != nil {
		t.Fatal(err)
	}

	application := bootstrap.NewApplication()
	packages, sources := buildCommandRegistries(t, application)
	err := executeBuildCommand(application, manifestPath, blockedOutput)
	if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}

	info, statErr := os.Stat(blockedOutput)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if !info.IsDir() {
		t.Fatalf("expected blocked output to remain a directory, got %v", info.Mode())
	}

	entries, err := os.ReadDir(blockedOutput)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no final package entries, got %#v", entries)
	}

	wantState := buildCommandRegistrySnapshot{
		packages: []registry.Package{{Name: "compiler", Version: "v1"}},
		sources: []compiler.PackageSource{
			{
				Name:       "compiler",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/compiler",
			},
		},
	}
	requireBuildCommandRegistrySnapshot(t, packages, sources, wantState)
}
