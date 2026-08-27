package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestRunnableOutputResolvesDefaultAbsolutePath(t *testing.T) {
	cwd := t.TempDir()
	got, err := resolveRunnableOutputPath(
		cwd,
		"",
		"demo",
		"v1",
		"windows",
		"amd64",
	)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cwd, "build", "demo-v1-runnable-windows-amd64.zip")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute output path, got %q", got)
	}
}

func TestRunnableOutputRejectsUnsafeDefaultComponents(t *testing.T) {
	cwd := t.TempDir()
	tests := map[string]struct {
		name, version, targetOS, targetArch string
	}{
		"empty name":           {version: "v1", targetOS: "linux", targetArch: "amd64"},
		"leading punctuation":  {name: "-demo", version: "v1", targetOS: "linux", targetArch: "amd64"},
		"trailing punctuation": {name: "demo", version: "v1.", targetOS: "linux", targetArch: "amd64"},
		"dot":                  {name: ".", version: "v1", targetOS: "linux", targetArch: "amd64"},
		"dot dot":              {name: "..", version: "v1", targetOS: "linux", targetArch: "amd64"},
		"path separator":       {name: "demo/app", version: "v1", targetOS: "linux", targetArch: "amd64"},
		"backslash":            {name: `demo\app`, version: "v1", targetOS: "linux", targetArch: "amd64"},
		"colon":                {name: "demo:app", version: "v1", targetOS: "linux", targetArch: "amd64"},
		"space":                {name: "demo app", version: "v1", targetOS: "linux", targetArch: "amd64"},
		"control":              {name: "demo\napp", version: "v1", targetOS: "linux", targetArch: "amd64"},
		"unsafe target OS":     {name: "demo", version: "v1", targetOS: "../linux", targetArch: "amd64"},
		"empty target arch":    {name: "demo", version: "v1", targetOS: "linux"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := resolveRunnableOutputPath(
				cwd,
				"",
				test.name,
				test.version,
				test.targetOS,
				test.targetArch,
			)
			requireRunnableOutputError(t, err)
		})
	}
}

func TestRunnableOutputAcceptsPortableDefaultComponents(t *testing.T) {
	got, err := resolveRunnableOutputPath(
		t.TempDir(),
		"",
		"Demo_App+Core",
		"v1.2-rc1",
		"linux",
		"arm64",
	)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "Demo_App+Core-v1.2-rc1-runnable-linux-arm64.zip" {
		t.Fatalf("unexpected default filename %q", filepath.Base(got))
	}
}

func TestRunnableOutputCustomPathContract(t *testing.T) {
	cwd := t.TempDir()
	relative := filepath.Join("dist", "custom.zip")
	got, err := resolveRunnableOutputPath(cwd, relative, "ignored/name", "ignored", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(cwd, relative) {
		t.Fatalf("expected exact relative resolution %q, got %q", filepath.Join(cwd, relative), got)
	}

	absolute := filepath.Join(t.TempDir(), "exact.zip")
	got, err = resolveRunnableOutputPath(cwd, absolute, "ignored", "ignored", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != absolute {
		t.Fatalf("expected exact absolute path %q, got %q", absolute, got)
	}

	for _, requested := range []string{"output", "output.ZIP", "output.zip ", "output.tar"} {
		t.Run(requested, func(t *testing.T) {
			_, err := resolveRunnableOutputPath(cwd, requested, "demo", "v1", "linux", "amd64")
			requireRunnableOutputError(t, err)
		})
	}
}

func TestRunnableOutputWhitespaceRequestUsesDefault(t *testing.T) {
	cwd := t.TempDir()
	got, err := resolveRunnableOutputPath(cwd, " \t\r\n ", "demo", "v1", "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cwd, "build", "demo-v1-runnable-linux-amd64.zip")
	if got != want {
		t.Fatalf("expected default output %q, got %q", want, got)
	}
}

func TestRunnableOutputStageCreatesPrivateSameFilesystemStaging(t *testing.T) {
	finalPath := filepath.Join(t.TempDir(), "nested", "output.zip")
	stage, err := prepareRunnableOutputStage(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := stage.Cleanup(); err != nil {
			t.Fatal(err)
		}
	}()

	if filepath.Dir(stage.directory) != filepath.Dir(finalPath) {
		t.Fatalf("staging directory %q is not under final parent %q", stage.directory, filepath.Dir(finalPath))
	}
	if stage.packagePath != filepath.Join(stage.directory, runnableStagedPackageName) {
		t.Fatalf("unexpected controlled staged package path %q", stage.packagePath)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(stage.directory)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("staging directory is not private: %v", info.Mode().Perm())
		}
	}
}

func TestRunnableOutputStageRejectsExistingObjects(t *testing.T) {
	for name, create := range map[string]func(*testing.T, string){
		"file": func(t *testing.T, path string) {
			if err := os.WriteFile(path, []byte("keep"), 0o644); err != nil {
				t.Fatal(err)
			}
		},
		"directory": func(t *testing.T, path string) {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "existing.zip")
			create(t, path)
			stage, err := prepareRunnableOutputStage(path)
			if stage != nil {
				t.Fatal("expected no stage for existing output")
			}
			requireRunnableOutputError(t, err)
		})
	}

	t.Run("symbolic link", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target")
		if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(directory, "existing.zip")
		if err := os.Symlink(target, link); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("Windows symlink creation is unavailable: %v", err)
			}
			t.Fatal(err)
		}
		stage, err := prepareRunnableOutputStage(link)
		if stage != nil {
			t.Fatal("expected no stage for existing symlink")
		}
		requireRunnableOutputError(t, err)
	})
}

func TestRunnablePublicationIsAtomicNoReplaceAndSurvivesCleanup(t *testing.T) {
	finalPath := filepath.Join(t.TempDir(), "published.zip")
	stage, err := prepareRunnableOutputStage(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("verified staged package")
	if err := os.WriteFile(stage.packagePath, payload, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := stage.Publish(); err != nil {
		t.Fatal(err)
	}
	if !stage.published {
		t.Fatal("expected publication state")
	}
	stagedInfo, err := os.Stat(stage.packagePath)
	if err != nil {
		t.Fatal(err)
	}
	finalInfo, err := os.Stat(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(stagedInfo, finalInfo) {
		t.Fatal("published path does not identify staged package")
	}

	if err := stage.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if err := stage.Cleanup(); err != nil {
		t.Fatalf("cleanup is not idempotent: %v", err)
	}
	got, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("published bytes changed: got %q", got)
	}
}

func TestRunnablePublicationRaceNeverOverwritesFinalTarget(t *testing.T) {
	finalPath := filepath.Join(t.TempDir(), "race.zip")
	stage, err := prepareRunnableOutputStage(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := stage.Cleanup(); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.WriteFile(stage.packagePath, []byte("staged"), 0o600); err != nil {
		t.Fatal(err)
	}
	original := []byte("existing target must survive")
	if err := os.WriteFile(finalPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	err = stage.Publish()
	requireRunnableOutputError(t, err)
	got, readErr := os.ReadFile(finalPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("publication overwrote final target: got %q", got)
	}
	if _, statErr := os.Stat(stage.packagePath); statErr != nil {
		t.Fatalf("staged package must remain for cleanup: %v", statErr)
	}
}

func TestRunnableOutputStageCleanupAfterFailedPublication(t *testing.T) {
	finalPath := filepath.Join(t.TempDir(), "missing-stage.zip")
	stage, err := prepareRunnableOutputStage(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	requireRunnableOutputError(t, stage.Publish())
	directory := stage.directory
	if err := stage.Cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("expected staging directory removal, got %v", err)
	}
	if _, err := os.Stat(finalPath); !os.IsNotExist(err) {
		t.Fatalf("failed publication created final output: %v", err)
	}
}

func TestRunnableOutputRejectsNonDirectoryParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(parent, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	stage, err := prepareRunnableOutputStage(filepath.Join(parent, "output.zip"))
	if stage != nil {
		t.Fatal("expected no stage for non-directory parent")
	}
	requireRunnableOutputError(t, err)
}

func requireRunnableOutputError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}
}
