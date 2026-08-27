package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func resolveRunPackagePath(cwd string, requested string) (string, error) {
	if strings.TrimSpace(cwd) == "" {
		return "", runPackageInputError("working directory is required", nil)
	}

	trimmedRequested := strings.TrimSpace(requested)
	if trimmedRequested == "" {
		return "", runPackageInputError("package path is required", nil)
	}
	if trimmedRequested != requested {
		return "", runPackageInputError("package path must not contain surrounding whitespace", nil)
	}
	if strings.Contains(requested, "://") {
		return "", runPackageInputError("package path must reference the local filesystem", nil)
	}
	if filepath.Ext(requested) != ".zip" {
		return "", runPackageInputError("package path must have exact .zip extension", nil)
	}

	absoluteCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", runPackageInputError("resolve working directory", err)
	}
	cwdInfo, err := os.Stat(absoluteCWD)
	if err != nil {
		return "", runPackageInputError("stat working directory", err)
	}
	if !cwdInfo.IsDir() {
		return "", runPackageInputError("working directory is not a directory", nil)
	}

	packagePath := requested
	if !filepath.IsAbs(packagePath) {
		packagePath = filepath.Join(absoluteCWD, packagePath)
	}
	absolutePath, err := filepath.Abs(filepath.Clean(packagePath))
	if err != nil {
		return "", runPackageInputError("resolve package path", err)
	}
	if filepath.Ext(absolutePath) != ".zip" {
		return "", runPackageInputError("package path must have exact .zip extension", nil)
	}

	return filepath.Clean(absolutePath), nil
}

// preflightRunPackageFile validates predictable local-file semantics only.
// Cryptographic and runtime content authority remains with
// runtime.VerifiedRunnablePackageLoader, which reopens the path later.
func preflightRunPackageFile(path string) error {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return runPackageInputError("package path must be absolute", nil)
	}
	if filepath.Ext(path) != ".zip" {
		return runPackageInputError("package path must have exact .zip extension", nil)
	}

	pathInfo, err := os.Lstat(path)
	if err != nil {
		return runPackageInputError("inspect package path", err)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 {
		return runPackageInputError("package path is a symbolic link", nil)
	}
	if !pathInfo.Mode().IsRegular() {
		return runPackageInputError("package path is not a regular file", nil)
	}

	file, err := os.Open(path)
	if err != nil {
		return runPackageInputError("open package file", err)
	}
	openInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return runPackageInputError("stat open package file", err)
	}
	if !openInfo.Mode().IsRegular() {
		_ = file.Close()
		return runPackageInputError("package path is not a regular file", nil)
	}
	if !os.SameFile(pathInfo, openInfo) {
		_ = file.Close()
		return runPackageInputError("package path identity changed while opening", nil)
	}
	if err := file.Close(); err != nil {
		return runPackageInputError("close package file", err)
	}

	return nil
}

func runPackageInputError(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", compiler.ErrInvalidArtifactPackage, message)
	}

	return fmt.Errorf("%w: %s: %w", compiler.ErrInvalidArtifactPackage, message, cause)
}
