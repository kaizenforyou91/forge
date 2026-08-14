package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileArtifactWriter persists artifacts into a deterministic filesystem tree.
type FileArtifactWriter struct {
	root string
}

// NewFileArtifactWriter creates a filesystem-backed artifact writer.
func NewFileArtifactWriter(root string) (*FileArtifactWriter, error) {
	if strings.TrimSpace(root) == "" {
		return nil, ErrInvalidOutputRoot
	}

	return &FileArtifactWriter{
		root: filepath.Clean(root),
	}, nil
}

// Write persists one artifact.
//
// Artifact output path:
//
//	<root>/<module>/<version>/artifact
//
// Module and version must be safe single path components.
func (w *FileArtifactWriter) Write(
	artifact Artifact,
	data []byte,
) error {
	if w == nil {
		return ErrNilArtifactWriter
	}

	if artifact.Module == "" || artifact.Version == "" {
		return ErrInvalidArtifact
	}

	if !validArtifactComponent(artifact.Module) ||
		!validArtifactComponent(artifact.Version) {
		return fmt.Errorf(
			"%w: unsafe artifact identity %q@%q",
			ErrInvalidArtifact,
			artifact.Module,
			artifact.Version,
		)
	}

	directory := filepath.Join(
		w.root,
		artifact.Module,
		artifact.Version,
	)

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}

	path := filepath.Join(directory, "artifact")

	if err := writeFileAtomic(path, data, 0o644); err != nil {
		return fmt.Errorf("write artifact %q: %w", path, err)
	}

	return nil
}

func validArtifactComponent(value string) bool {
	if value == "." || value == ".." {
		return false
	}

	if filepath.Base(value) != value {
		return false
	}

	if strings.ContainsAny(value, `/\`) {
		return false
	}

	return true
}

func writeFileAtomic(
	path string,
	data []byte,
	perm os.FileMode,
) error {
	directory := filepath.Dir(path)

	file, err := os.CreateTemp(directory, ".artifact-*")
	if err != nil {
		return err
	}

	tempPath := file.Name()

	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(tempPath)
	}

	if _, err := file.Write(data); err != nil {
		cleanup()
		return err
	}

	if err := file.Sync(); err != nil {
		cleanup()
		return err
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if err := os.Chmod(tempPath, perm); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}

	return nil
}
