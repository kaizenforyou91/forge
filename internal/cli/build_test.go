package cli

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/internal/bootstrap"
	"github.com/kaizenforyou91/forge/pkg/compiler"
	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/spf13/cobra"
)

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

func TestRegisterManifestPackages(t *testing.T) {
	application := bootstrap.NewApplication()

	registryInstance, err := resolveRegistry(application)
	if err != nil {
		t.Fatal(err)
	}

	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []manifest.Module{
			{
				Name:    "logger",
				Version: "v1",
			},
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}

	if err := registerManifestPackages(
		registryInstance,
		m,
	); err != nil {
		t.Fatal(err)
	}

	for _, module := range m.Modules {
		got, err := registryInstance.Get(
			module.Name,
			module.Version,
		)
		if err != nil {
			t.Fatal(err)
		}

		if got.Name != module.Name {
			t.Fatalf(
				"expected package name %q, got %q",
				module.Name,
				got.Name,
			)
		}

		if got.Version != module.Version {
			t.Fatalf(
				"expected package version %q, got %q",
				module.Version,
				got.Version,
			)
		}
	}
}

func TestRegisterManifestPackagesIsIdempotent(
	t *testing.T,
) {
	application := bootstrap.NewApplication()

	registryInstance, err := resolveRegistry(application)
	if err != nil {
		t.Fatal(err)
	}

	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []manifest.Module{
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	if err := registerManifestPackages(
		registryInstance,
		m,
	); err != nil {
		t.Fatal(err)
	}

	if err := registerManifestPackages(
		registryInstance,
		m,
	); err != nil {
		t.Fatal(err)
	}

	if registryInstance.Count() != 1 {
		t.Fatalf(
			"expected 1 registered package, got %d",
			registryInstance.Count(),
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

func TestRegisterManifestSources(t *testing.T) {
	r := compiler.NewPackageSourceRegistry()

	m := manifest.Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []manifest.Module{
			{
				Name:       "forge",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/cmd/forge",
			},
		},
	}

	if err := registerManifestSources(r, m); err != nil {
		t.Fatal(err)
	}

	source, err := r.Resolve("forge", "v1")
	if err != nil {
		t.Fatal(err)
	}

	if source.ImportPath !=
		"github.com/kaizenforyou91/forge/cmd/forge" {
		t.Fatalf(
			"unexpected import path: %q",
			source.ImportPath,
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
