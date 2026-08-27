package cli

import (
	"bytes"
	"errors"
	"net"
	"os"
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

	_, err := resolveRunPackagePath(filepath.Join(t.TempDir(), "missing"), "application.zip")
	requireRunPackageInputError(t, err)

	nonDirectory := filepath.Join(t.TempDir(), "cwd-file")
	if err := os.WriteFile(nonDirectory, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = resolveRunPackagePath(nonDirectory, "application.zip")
	requireRunPackageInputError(t, err)
}

func TestRunPackageFilePreflightAcceptsRegularLocalFileWithoutReadingContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application.zip")
	content := []byte("content validation belongs to the runtime loader")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := preflightRunPackageFile(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatal("package preflight changed package contents")
	}
}

func TestRunPackageFilePreflightRejectsInvalidLocalObjects(t *testing.T) {
	t.Run("relative path", func(t *testing.T) {
		requireRunPackageInputError(t, preflightRunPackageFile("application.zip"))
	})
	t.Run("non-zip", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "application.bin")
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		requireRunPackageInputError(t, preflightRunPackageFile(path))
	})
	t.Run("missing file", func(t *testing.T) {
		requireRunPackageInputError(
			t,
			preflightRunPackageFile(filepath.Join(t.TempDir(), "missing.zip")),
		)
	})
	t.Run("directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "directory.zip")
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		requireRunPackageInputError(t, preflightRunPackageFile(path))
	})
	t.Run("symbolic link", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target.zip")
		if err := os.WriteFile(target, []byte("package"), 0o644); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(directory, "link.zip")
		if err := os.Symlink(target, link); err != nil {
			if runtime.GOOS == "windows" {
				t.Skipf("Windows symlink creation is unavailable: %v", err)
			}
			t.Fatal(err)
		}
		requireRunPackageInputError(t, preflightRunPackageFile(link))
	})
	t.Run("socket", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Unix-domain filesystem socket test is not available on Windows")
		}
		path := filepath.Join(t.TempDir(), "socket.zip")
		listener, err := net.Listen("unix", path)
		if err != nil {
			t.Skipf("Unix-domain socket creation is unavailable: %v", err)
		}
		defer listener.Close()
		requireRunPackageInputError(t, preflightRunPackageFile(path))
	})
}

func requireRunPackageInputError(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, compiler.ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}
}
