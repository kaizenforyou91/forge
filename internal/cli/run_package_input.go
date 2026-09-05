package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

func resolveRunPackagePath(cwd string, requested string) (string, error) {
	return resolveLocalPackagePath(cwd, requested)
}

func resolveLocalPackagePath(cwd string, requested string) (string, error) {
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

func runPackageInputError(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", compiler.ErrInvalidArtifactPackage, message)
	}

	return fmt.Errorf("%w: %s: %w", compiler.ErrInvalidArtifactPackage, message, cause)
}
