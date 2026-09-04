package cli

import (
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
	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

func TestValidateCommandProfiles(t *testing.T) {
	tests := []struct {
		name        string
		profileArgs []string
		wantProfile string
	}{
		{name: "default build", wantProfile: "build"},
		{name: "explicit structural", profileArgs: []string{"--profile", "structural"}, wantProfile: "structural"},
		{name: "explicit build", profileArgs: []string{"--profile", "build"}, wantProfile: "build"},
		{name: "explicit runnable", profileArgs: []string{"--profile", "runnable"}, wantProfile: "runnable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			manifestPath := writeValidateManifest(
				t,
				directory,
				"forge.yaml",
				[]byte(validRunnableValidationYAML("example.invalid/no-toolchain")),
			)
			application, packages, sources := newValidationOnlyApplication(t)
			before := snapshotBuildCommandRegistries(packages, sources)

			stdout, err := executeValidateCommand(
				application,
				manifestPath,
				test.profileArgs...,
			)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintf(
				"Manifest valid: %s (profile=%s)\n",
				manifestPath,
				test.wantProfile,
			)
			if stdout != want {
				t.Fatalf("expected output %q, got %q", want, stdout)
			}
			requireBuildCommandRegistrySnapshot(t, packages, sources, before)
			requireValidateDirectoryContainsOnly(t, directory, "forge.yaml")
		})
	}
}

func TestValidateCommandRejectsNonExactProfilesWithoutSideEffects(t *testing.T) {
	invalidProfiles := []string{"BUILD", "Build", " build", "build ", "unknown", ""}

	for _, profile := range invalidProfiles {
		name := profile
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			manifestPath := writeValidateManifest(
				t,
				directory,
				"forge.yaml",
				[]byte(validBuildValidationYAML("example.com/app")),
			)
			application, packages, sources := newValidationOnlyApplication(t)
			seedBuildCommandRegistries(t, packages, sources)
			before := snapshotBuildCommandRegistries(packages, sources)
			flag := "--profile=" + profile

			stdout, err := executeValidateCommand(application, manifestPath, flag)
			if err == nil {
				t.Fatal("expected invalid profile error")
			}
			if !strings.Contains(err.Error(), "invalid validation profile") {
				t.Fatalf("unexpected error: %v", err)
			}
			if stdout != "" {
				t.Fatalf("expected no success output, got %q", stdout)
			}
			requireBuildCommandRegistrySnapshot(t, packages, sources, before)
			requireValidateDirectoryContainsOnly(t, directory, "forge.yaml")
		})
	}
}

func TestValidateStructuralProfileSupportsStrictYAMLAndJSON(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		data     []byte
	}{
		{
			name:     "YAML",
			filename: "forge.yaml",
			data:     []byte(validBuildValidationYAML("example.com/app")),
		},
		{
			name:     "JSON",
			filename: "forge.json",
			data:     []byte(validBuildValidationJSON("example.com/app", false)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			manifestPath := writeValidateManifest(t, directory, test.filename, test.data)
			stdout, err := executeValidateCommand(
				app.New(),
				manifestPath,
				"--profile",
				"structural",
			)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(stdout, "(profile=structural)") {
				t.Fatalf("unexpected output %q", stdout)
			}
		})
	}
}

func TestValidateStructuralProfileRejectsParserAndDomainFailures(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		data     []byte
	}{
		{name: "unknown field", filename: "forge.yaml", data: []byte("version: v1\nname: demo\nmodules: []\nunknown: value\n")},
		{name: "duplicate JSON key", filename: "forge.json", data: []byte(`{"version":"v1","name":"demo","name":"other","modules":[]}`)},
		{name: "forbidden YAML anchor", filename: "forge.yaml", data: []byte("version: &version v1\nname: demo\nmodules: []\n")},
		{name: "ambiguous identity", filename: "forge.json", data: []byte(`{"version":"v1","name":"demo@other","modules":[]}`)},
		{name: "missing required name", filename: "forge.json", data: []byte(`{"version":"v1","modules":[]}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			manifestPath := writeValidateManifest(t, directory, test.filename, test.data)
			application, packages, sources := newValidationOnlyApplication(t)
			seedBuildCommandRegistries(t, packages, sources)
			before := snapshotBuildCommandRegistries(packages, sources)

			stdout, err := executeValidateCommand(
				application,
				manifestPath,
				"--profile",
				"structural",
			)
			requireInvalidManifestCode(t, err)
			if stdout != "" {
				t.Fatalf("expected no success output, got %q", stdout)
			}
			requireBuildCommandRegistrySnapshot(t, packages, sources, before)
			requireValidateDirectoryContainsOnly(t, directory, test.filename)
		})
	}
}

func TestValidateAdmissionProfilesRunDomainValidationBeforeServiceResolution(t *testing.T) {
	for _, profile := range []string{"build", "runnable"} {
		t.Run(profile, func(t *testing.T) {
			path := writeValidateManifest(
				t,
				t.TempDir(),
				"forge.json",
				[]byte(`{"version":"v1","name":"demo@other","modules":[]}`),
			)
			_, err := executeValidateCommand(
				app.New(),
				path,
				"--profile",
				profile,
			)
			requireInvalidManifestCode(t, err)
			if strings.Contains(err.Error(), "service") {
				t.Fatalf("service resolution preceded domain validation: %v", err)
			}
		})
	}
}

func TestValidateBuildProfileUsesNonMutatingPreparedAdmission(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		setup   func(*testing.T, *registry.Registry, *compiler.PackageSourceRegistry)
		wantErr func(*testing.T, error)
	}{
		{
			name: "canonical import path",
			data: validBuildValidationYAML(" example.com/app "),
		},
		{
			name: "missing dependency",
			data: "version: v1\nname: demo\nmodules:\n  - name: app\n    version: v1\n    import_path: example.com/app\n    dependencies:\n      - name: missing\n        version: v1\n",
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				requireForgeErrorCode(t, err, forgeerrors.CodeNotFound)
			},
		},
		{
			name: "dependency cycle",
			data: "version: v1\nname: demo\nmodules:\n  - name: app\n    version: v1\n    import_path: example.com/app\n    dependencies: [{name: dep, version: v1}]\n  - name: dep\n    version: v1\n    import_path: example.com/dep\n    dependencies: [{name: app, version: v1}]\n",
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				requireForgeErrorCode(t, err, forgeerrors.CodeInvalidManifest)
			},
		},
		{
			name: "invalid import path",
			data: validBuildValidationYAML("   "),
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, compiler.ErrInvalidPackageSource) {
					t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
				}
			},
		},
		{
			name: "source conflict",
			data: validBuildValidationYAML("example.com/requested"),
			setup: func(t *testing.T, packages *registry.Registry, sources *compiler.PackageSourceRegistry) {
				t.Helper()
				if err := packages.EnsureAll([]registry.Package{{Name: "app", Version: "v1"}}); err != nil {
					t.Fatal(err)
				}
				if err := sources.EnsureAll([]compiler.PackageSource{{Name: "app", Version: "v1", ImportPath: "example.com/canonical"}}); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				if !errors.Is(err, compiler.ErrPackageSourceConflict) {
					t.Fatalf("expected ErrPackageSourceConflict, got %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			manifestPath := writeValidateManifest(t, directory, "forge.yaml", []byte(test.data))
			application, packages, sources := newValidationOnlyApplication(t)
			if test.setup != nil {
				test.setup(t, packages, sources)
			}
			before := snapshotBuildCommandRegistries(packages, sources)

			stdout, err := executeValidateCommand(application, manifestPath, "--profile", "build")
			if test.wantErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(stdout, "(profile=build)") {
					t.Fatalf("unexpected output %q", stdout)
				}
			} else {
				if err == nil {
					t.Fatal("expected build validation failure")
				}
				test.wantErr(t, err)
				if stdout != "" {
					t.Fatalf("expected no success output, got %q", stdout)
				}
			}

			requireBuildCommandRegistrySnapshot(t, packages, sources, before)
			requireValidateDirectoryContainsOnly(t, directory, "forge.yaml")
		})
	}
}

func TestValidateRunnableProfileUsesAdmissionWithoutToolchainOrSigning(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr error
	}{
		{
			name: "canonical admitted entrypoint",
			data: validRunnableValidationYAML("example.invalid/not-buildable"),
		},
		{
			name:    "missing entrypoint",
			data:    validBuildValidationYAML("example.com/app"),
			wantErr: compiler.ErrInvalidApplicationEntrypoint,
		},
		{
			name:    "invalid admitted entrypoint source",
			data:    validRunnableValidationYAML("   "),
			wantErr: compiler.ErrInvalidPackageSource,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			manifestPath := writeValidateManifest(t, directory, "forge.yaml", []byte(test.data))
			application, packages, sources := newValidationOnlyApplication(t)
			before := snapshotBuildCommandRegistries(packages, sources)

			stdout, err := executeValidateCommand(application, manifestPath, "--profile", "runnable")
			if test.wantErr == nil {
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(stdout, "(profile=runnable)") {
					t.Fatalf("unexpected output %q", stdout)
				}
			} else {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected %v, got %v", test.wantErr, err)
				}
				if stdout != "" {
					t.Fatalf("expected no success output, got %q", stdout)
				}
			}

			requireBuildCommandRegistrySnapshot(t, packages, sources, before)
			requireValidateDirectoryContainsOnly(t, directory, "forge.yaml")
		})
	}
}

func TestValidateAdmissionProfilesDoNotMutateManifestValue(t *testing.T) {
	for _, profile := range []manifestValidationProfile{
		manifestValidationProfileBuild,
		manifestValidationProfileRunnable,
	} {
		t.Run(string(profile), func(t *testing.T) {
			application, _, _ := newValidationOnlyApplication(t)
			cmd := newValidateCmd()
			cmd.SetContext(NewApplicationContext(application))
			m := manifest.Manifest{
				Version: "v1",
				Name:    "demo",
				Entrypoint: &manifest.ApplicationEntrypoint{
					Module:  "app",
					Version: "v1",
				},
				Modules: []manifest.Module{
					{
						Name:       "app",
						Version:    "v1",
						ImportPath: " example.invalid/no-toolchain ",
						Dependencies: []manifest.Dependency{
							{Name: "dep", Version: "v1"},
						},
					},
					{Name: "dep", Version: "v1", ImportPath: " example.invalid/dep "},
				},
			}
			want := manifest.Manifest{
				Version: "v1",
				Name:    "demo",
				Entrypoint: &manifest.ApplicationEntrypoint{
					Module:  "app",
					Version: "v1",
				},
				Modules: []manifest.Module{
					{
						Name:       "app",
						Version:    "v1",
						ImportPath: " example.invalid/no-toolchain ",
						Dependencies: []manifest.Dependency{
							{Name: "dep", Version: "v1"},
						},
					},
					{Name: "dep", Version: "v1", ImportPath: " example.invalid/dep "},
				},
			}

			if err := validateManifestForProfile(cmd, m, profile); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(m, want) {
				t.Fatalf("manifest mutated:\nwant: %#v\ngot:  %#v", want, m)
			}
		})
	}
}

func TestValidateCommandPropagatesSuccessOutputFailure(t *testing.T) {
	directory := t.TempDir()
	manifestPath := writeValidateManifest(
		t,
		directory,
		"forge.yaml",
		[]byte(validBuildValidationYAML("example.com/app")),
	)
	wantErr := errors.New("write failed")
	cmd := NewRootCommandWithApplication(app.New())
	cmd.SetArgs([]string{"validate", manifestPath, "--profile", "structural"})
	cmd.SetOut(validateErrorWriter{err: wantErr})
	cmd.SetErr(io.Discard)

	err := cmd.Execute()
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected writer failure, got %v", err)
	}
	requireValidateDirectoryContainsOnly(t, directory, "forge.yaml")
}

func TestValidateCommandInheritsManifestExtensionSelection(t *testing.T) {
	tests := []struct {
		filename string
		data     []byte
	}{
		{filename: "forge.yaml", data: []byte(validBuildValidationYAML("example.com/app"))},
		{filename: "forge.yml", data: []byte(validBuildValidationYAML("example.com/app"))},
		{filename: "forge.YAML", data: []byte(validBuildValidationYAML("example.com/app"))},
		{filename: "forge.YML", data: []byte(validBuildValidationYAML("example.com/app"))},
		{filename: "forge.json", data: []byte(validBuildValidationJSON("example.com/app", false))},
		{filename: "forge.JSON", data: []byte(validBuildValidationJSON("example.com/app", false))},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			path := writeValidateManifest(t, t.TempDir(), test.filename, test.data)
			if _, err := executeValidateCommand(app.New(), path, "--profile", "structural"); err != nil {
				t.Fatal(err)
			}
		})
	}

	directory := t.TempDir()
	path := writeValidateManifest(t, directory, "forge.toml", []byte("version = 'v1'"))
	stdout, err := executeValidateCommand(app.New(), path, "--profile", "structural")
	if err == nil || !strings.Contains(err.Error(), "unsupported manifest format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
	if stdout != "" {
		t.Fatalf("expected no success output, got %q", stdout)
	}
}

func TestValidateCommandPreservesFilesystemCause(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")
	_, err := executeValidateCommand(app.New(), path, "--profile", "structural")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestValidateAndBuildShareStrictLoaderRejection(t *testing.T) {
	directory := t.TempDir()
	manifestPath := writeValidateManifest(
		t,
		directory,
		"forge.json",
		[]byte(`{"version":"v1","name":"demo","modules":[],"modules":[]}`),
	)

	validateApplication, validatePackages, validateSources := newValidationOnlyApplication(t)
	validateBefore := snapshotBuildCommandRegistries(validatePackages, validateSources)
	validateOutput, validateErr := executeValidateCommand(
		validateApplication,
		manifestPath,
		"--profile",
		"build",
	)
	requireInvalidManifestCode(t, validateErr)
	if validateOutput != "" {
		t.Fatalf("expected no validate output, got %q", validateOutput)
	}
	requireBuildCommandRegistrySnapshot(t, validatePackages, validateSources, validateBefore)

	buildApplication := bootstrap.NewApplication()
	buildPackages, buildSources := buildCommandRegistries(t, buildApplication)
	buildBefore := snapshotBuildCommandRegistries(buildPackages, buildSources)
	outputPath := filepath.Join(directory, "must-not-exist.zip")
	buildErr := executeBuildCommand(buildApplication, manifestPath, outputPath)
	requireInvalidManifestCode(t, buildErr)
	requireBuildCommandRegistrySnapshot(t, buildPackages, buildSources, buildBefore)
	requireBuildCommandOutputAbsent(t, outputPath)
}

func TestValidateRunnableAndBuildRunnableShareAdmissionFailure(t *testing.T) {
	directory := t.TempDir()
	manifestPath := writeValidateManifest(
		t,
		directory,
		"forge.yaml",
		[]byte(validBuildValidationYAML("example.com/app")),
	)

	validateApplication, validatePackages, validateSources := newValidationOnlyApplication(t)
	validateBefore := snapshotBuildCommandRegistries(validatePackages, validateSources)
	validateOutput, validateErr := executeValidateCommand(
		validateApplication,
		manifestPath,
		"--profile",
		"runnable",
	)
	if !errors.Is(validateErr, compiler.ErrInvalidApplicationEntrypoint) {
		t.Fatalf("expected validate entrypoint error, got %v", validateErr)
	}
	if validateOutput != "" {
		t.Fatalf("expected no validate output, got %q", validateOutput)
	}
	requireBuildCommandRegistrySnapshot(t, validatePackages, validateSources, validateBefore)

	buildApplication := bootstrap.NewApplication()
	buildPackages, buildSources := buildCommandRegistries(t, buildApplication)
	buildBefore := snapshotBuildCommandRegistries(buildPackages, buildSources)
	outputPath := filepath.Join(directory, "must-not-exist.zip")
	_, buildErr := executeBuildRunnableCommand(
		buildApplication,
		nil,
		manifestPath,
		filepath.Join(directory, "missing-key.pem"),
		buildRunnableTestKeyID,
		outputPath,
	)
	if !errors.Is(buildErr, compiler.ErrInvalidApplicationEntrypoint) {
		t.Fatalf("expected build-runnable entrypoint error, got %v", buildErr)
	}
	requireBuildCommandRegistrySnapshot(t, buildPackages, buildSources, buildBefore)
	requireBuildCommandOutputAbsent(t, outputPath)
}

func TestValidateCommandArgumentAndHelpContract(t *testing.T) {
	for name, args := range map[string][]string{
		"missing manifest":      {"validate"},
		"too many manifests":    {"validate", "one.yaml", "two.yaml"},
		"missing profile value": {"validate", "manifest.yaml", "--profile"},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := NewRootCommand()
			cmd.SetArgs(args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected CLI grammar error")
			}
		})
	}

	cmd := NewRootCommand()
	var output strings.Builder
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"validate", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"validate <manifest>",
		"--profile",
		"structural",
		"build",
		"runnable",
		"does not prove compilation",
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("expected help to contain %q, got %q", expected, output.String())
		}
	}
}

type validateErrorWriter struct {
	err error
}

func (writer validateErrorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func newValidationOnlyApplication(
	t *testing.T,
) (*app.App, *registry.Registry, *compiler.PackageSourceRegistry) {
	t.Helper()
	application := app.New()
	packages := registry.New()
	sources := compiler.NewPackageSourceRegistry()
	if err := application.Container().RegisterSingleton(packages); err != nil {
		t.Fatal(err)
	}
	if err := application.Container().RegisterSingleton(sources); err != nil {
		t.Fatal(err)
	}
	return application, packages, sources
}

func executeValidateCommand(
	application *app.App,
	manifestPath string,
	extraArgs ...string,
) (string, error) {
	cmd := NewRootCommandWithApplication(application)
	args := []string{"validate", manifestPath}
	args = append(args, extraArgs...)
	cmd.SetArgs(args)
	var output strings.Builder
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	err := cmd.Execute()
	return output.String(), err
}

func writeValidateManifest(
	t *testing.T,
	directory,
	filename string,
	data []byte,
) string {
	t.Helper()
	path := filepath.Join(directory, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func validBuildValidationYAML(importPath string) string {
	return fmt.Sprintf(
		"version: v1\nname: demo\nmodules:\n  - name: app\n    version: v1\n    import_path: %q\n",
		importPath,
	)
}

func validRunnableValidationYAML(importPath string) string {
	return fmt.Sprintf(
		"version: v1\nname: demo\nentrypoint:\n  module: app\n  version: v1\nmodules:\n  - name: app\n    version: v1\n    import_path: %q\n",
		importPath,
	)
}

func validBuildValidationJSON(importPath string, withEntrypoint bool) string {
	entrypoint := ""
	if withEntrypoint {
		entrypoint = `"entrypoint":{"module":"app","version":"v1"},`
	}
	return fmt.Sprintf(
		`{"version":"v1","name":"demo",%s"modules":[{"name":"app","version":"v1","import_path":%q}]}`,
		entrypoint,
		importPath,
	)
}

func requireForgeErrorCode(t *testing.T, err error, want forgeerrors.Code) {
	t.Helper()
	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf("expected *errors.Error, got %T: %v", err, err)
	}
	if forgeErr.Code != want {
		t.Fatalf("expected error code %s, got %s", want, forgeErr.Code)
	}
}

func requireValidateDirectoryContainsOnly(
	t *testing.T,
	directory,
	manifestName string,
) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(entries))
	for i, entry := range entries {
		got[i] = entry.Name()
	}
	if want := []string{manifestName}; !reflect.DeepEqual(got, want) {
		t.Fatalf("validation created filesystem output: want %#v, got %#v", want, got)
	}
}
