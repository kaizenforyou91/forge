package cli

import (
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func TestRunPackagePathResolvesRelativeAndAbsoluteZIPPaths(t *testing.T) {
	cwd := t.TempDir()
	relative := filepath.Join("packages", "application.zip")

	got, err := resolveRunPackagePath(cwd, relative)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cwd, relative)
	want, err = filepath.Abs(want)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("expected %q, got %q", filepath.Clean(want), got)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute package path, got %q", got)
	}

	missingCWD := filepath.Join(t.TempDir(), "not-created")
	got, err = resolveRunPackagePath(missingCWD, "missing.zip")
	if err != nil {
		t.Fatalf("lexical resolver inspected package existence: %v", err)
	}
	want = filepath.Join(missingCWD, "missing.zip")
	want, err = filepath.Abs(want)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("expected nonexistent lexical path %q, got %q", filepath.Clean(want), got)
	}

	absolute := filepath.Join(t.TempDir(), "absolute.zip")
	got, err = resolveRunPackagePath(cwd, absolute)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(absolute) {
		t.Fatalf("expected absolute path %q, got %q", filepath.Clean(absolute), got)
	}
}

func TestRunPackagePathRejectsInvalidInputs(t *testing.T) {
	cwd := t.TempDir()
	tests := map[string]string{
		"blank":                    "",
		"whitespace only":          " \t\r\n ",
		"leading whitespace":       " application.zip",
		"trailing whitespace":      "application.zip ",
		"missing extension":        "application",
		"uppercase extension":      "application.ZIP",
		"trailing extra extension": "application.zip.tmp",
		"remote URL":               "https://example.com/application.zip",
	}
	for name, requested := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := resolveRunPackagePath(cwd, requested)
			requireRunPackageInputError(t, err)
		})
	}

	if runtime.GOOS == "windows" {
		got, err := resolveRunPackagePath(cwd, `C:\packages\application.zip`)
		if err != nil {
			t.Fatalf("Windows drive path was treated as remote: %v", err)
		}
		if got != filepath.Clean(`C:\packages\application.zip`) {
			t.Fatalf("expected cleaned Windows drive path, got %q", got)
		}
	}
}

func requireRunPackageInputError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}
}
