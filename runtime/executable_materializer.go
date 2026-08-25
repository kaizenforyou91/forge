package runtime

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
)

const runtimeMaterializationDirectoryPattern = "forge-runtime-*"

// SecureExecutableMaterializer writes strictly verified executable bytes into
// a private, controlled filesystem location without executing them.
type SecureExecutableMaterializer struct{}

func NewSecureExecutableMaterializer() *SecureExecutableMaterializer {
	return &SecureExecutableMaterializer{}
}

// Materialize creates and validates one private host executable. The returned
// lifecycle object exclusively owns cleanup of the private directory.
func (m *SecureExecutableMaterializer) Materialize(
	pkg VerifiedRunnablePackage,
) (result *MaterializedExecutable, resultErr error) {
	executable, err := validateMaterializationInput(m, pkg)
	if err != nil {
		return nil, err
	}

	expectedSize := int64(len(executable))
	expectedSHA256 := sha256.Sum256(executable)

	directory, err := os.MkdirTemp("", runtimeMaterializationDirectoryPattern)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: create private directory: %w",
			ErrExecutableMaterializationFailed,
			err,
		)
	}

	cleanupRequired := true
	defer func() {
		if !cleanupRequired {
			return
		}

		if cleanupErr := os.RemoveAll(directory); cleanupErr != nil {
			wrapped := fmt.Errorf(
				"%w: cleanup private directory %q: %w",
				ErrExecutableMaterializationFailed,
				directory,
				cleanupErr,
			)
			if resultErr == nil {
				resultErr = wrapped
				return
			}

			resultErr = errors.Join(resultErr, wrapped)
		}
	}()

	directoryInfo, err := os.Lstat(directory)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: inspect private directory %q: %w",
			ErrExecutableMaterializationFailed,
			directory,
			err,
		)
	}
	if err := validatePrivateMaterializationDirectory(directory, directoryInfo); err != nil {
		return nil, err
	}

	executablePath := filepath.Join(directory, materializedExecutableFileName())
	file, err := os.OpenFile(
		executablePath,
		os.O_RDWR|os.O_CREATE|os.O_EXCL,
		0o600,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: create controlled executable %q: %w",
			ErrExecutableMaterializationFailed,
			executablePath,
			err,
		)
	}

	fileOpen := true
	defer func() {
		if !fileOpen {
			return
		}

		if closeErr := file.Close(); closeErr != nil {
			wrapped := fmt.Errorf(
				"%w: close controlled executable %q after failure: %w",
				ErrExecutableMaterializationFailed,
				executablePath,
				closeErr,
			)
			if resultErr == nil {
				resultErr = wrapped
				return
			}

			resultErr = errors.Join(resultErr, wrapped)
		}
	}()

	if err := writeAll(file, executable); err != nil {
		return nil, fmt.Errorf(
			"%w: write controlled executable %q: %w",
			ErrExecutableMaterializationFailed,
			executablePath,
			err,
		)
	}
	if err := file.Chmod(0o700); err != nil {
		return nil, fmt.Errorf(
			"%w: set controlled executable permissions %q: %w",
			ErrExecutableMaterializationFailed,
			executablePath,
			err,
		)
	}
	// Sync surfaces deferred write failures before the temporary executable is
	// handed off. It does not provide installation or crash-durability semantics.
	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf(
			"%w: sync controlled executable %q: %w",
			ErrExecutableMaterializationFailed,
			executablePath,
			err,
		)
	}

	openInfo, err := validateOpenMaterializedExecutable(
		file,
		executablePath,
		expectedSize,
		expectedSHA256,
	)
	if err != nil {
		return nil, err
	}
	if err := validateMaterializedExecutablePath(
		executablePath,
		openInfo,
		expectedSize,
	); err != nil {
		return nil, err
	}

	if err := file.Close(); err != nil {
		fileOpen = false
		return nil, fmt.Errorf(
			"%w: close controlled executable %q: %w",
			ErrExecutableMaterializationFailed,
			executablePath,
			err,
		)
	}
	fileOpen = false

	if err := validateMaterializedExecutablePath(
		executablePath,
		openInfo,
		expectedSize,
	); err != nil {
		return nil, err
	}

	result = &MaterializedExecutable{
		directory:      directory,
		path:           executablePath,
		expectedSize:   expectedSize,
		expectedSHA256: expectedSHA256,
		entrypoint:     pkg.Entrypoint(),
		signerKeyID:    pkg.SignerKeyID(),
		targetOS:       pkg.TargetOS(),
		targetArch:     pkg.TargetArch(),
		removeAll:      os.RemoveAll,
	}
	cleanupRequired = false

	return result, nil
}

func validateMaterializationInput(
	m *SecureExecutableMaterializer,
	pkg VerifiedRunnablePackage,
) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf(
			"%w: materializer is nil",
			ErrInvalidRunnablePackage,
		)
	}
	if pkg.PackageFormatVersion() != runnablePackageFormatVersion ||
		pkg.BundleSchemaVersion() != runnableBundleSchemaVersion {
		return nil, fmt.Errorf(
			"%w: expected package format %d / bundle schema %d, got %d / %d",
			ErrInvalidRunnablePackage,
			runnablePackageFormatVersion,
			runnableBundleSchemaVersion,
			pkg.PackageFormatVersion(),
			pkg.BundleSchemaVersion(),
		)
	}
	if strings.TrimSpace(pkg.ManifestName()) == "" ||
		strings.TrimSpace(pkg.ManifestVersion()) == "" {
		return nil, fmt.Errorf(
			"%w: manifest identity is incomplete",
			ErrInvalidRunnablePackage,
		)
	}

	entrypoint := pkg.Entrypoint()
	if strings.TrimSpace(entrypoint.Module) == "" ||
		strings.TrimSpace(entrypoint.Version) == "" {
		return nil, fmt.Errorf(
			"%w: entrypoint identity is incomplete",
			ErrInvalidRunnablePackage,
		)
	}
	if strings.TrimSpace(pkg.ImportPath()) == "" {
		return nil, fmt.Errorf(
			"%w: entrypoint import path is required",
			ErrInvalidRunnablePackage,
		)
	}
	if strings.TrimSpace(pkg.SignerKeyID()) == "" {
		return nil, fmt.Errorf(
			"%w: verified signer evidence is required",
			ErrInvalidRunnablePackage,
		)
	}
	if strings.TrimSpace(pkg.TargetOS()) == "" ||
		strings.TrimSpace(pkg.TargetArch()) == "" {
		return nil, fmt.Errorf(
			"%w: runtime target is incomplete",
			ErrInvalidRunnablePackage,
		)
	}
	if pkg.TargetOS() != goruntime.GOOS || pkg.TargetArch() != goruntime.GOARCH {
		return nil, fmt.Errorf(
			"%w: package targets %s/%s, host is %s/%s",
			ErrUnsupportedRuntimePlatform,
			pkg.TargetOS(),
			pkg.TargetArch(),
			goruntime.GOOS,
			goruntime.GOARCH,
		)
	}

	executable := pkg.ExecutableBytes()
	if len(executable) == 0 {
		return nil, fmt.Errorf(
			"%w: verified executable bytes are empty",
			ErrInvalidRunnablePackage,
		)
	}

	return executable, nil
}

func validatePrivateMaterializationDirectory(path string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf(
			"%w: private directory %q is a symbolic link",
			ErrExecutableMaterializationFailed,
			path,
		)
	}
	if !info.IsDir() {
		return fmt.Errorf(
			"%w: private directory %q is not a directory",
			ErrExecutableMaterializationFailed,
			path,
		)
	}
	if goruntime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf(
			"%w: private directory %q grants group or other permissions %04o",
			ErrExecutableMaterializationFailed,
			path,
			info.Mode().Perm(),
		)
	}

	return nil
}

func materializedExecutableFileName() string {
	if goruntime.GOOS == "windows" {
		return "application.exe"
	}

	return "application"
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if written < 0 || written > len(data) {
			return fmt.Errorf("invalid write count %d for %d remaining bytes", written, len(data))
		}
		if written > 0 {
			data = data[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
	}

	return nil
}

func validateOpenMaterializedExecutable(
	file *os.File,
	path string,
	expectedSize int64,
	expectedSHA256 [32]byte,
) (os.FileInfo, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf(
			"%w: seek controlled executable %q: %w",
			ErrExecutableMaterializationFailed,
			path,
			err,
		)
	}

	hasher := sha256.New()
	read, err := io.Copy(hasher, file)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: read back controlled executable %q: %w",
			ErrExecutableMaterializationFailed,
			path,
			err,
		)
	}
	if read != expectedSize {
		return nil, fmt.Errorf(
			"%w: read back %d bytes from %q, expected %d",
			ErrMaterializedExecutableInvalid,
			read,
			path,
			expectedSize,
		)
	}

	var actualSHA256 [32]byte
	copy(actualSHA256[:], hasher.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return nil, fmt.Errorf(
			"%w: SHA-256 mismatch for %q",
			ErrMaterializedExecutableInvalid,
			path,
		)
	}

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf(
			"%w: stat open controlled executable %q: %w",
			ErrExecutableMaterializationFailed,
			path,
			err,
		)
	}
	if err := validateMaterializedExecutableInfo(path, info, expectedSize); err != nil {
		return nil, err
	}

	return info, nil
}

func validateMaterializedExecutablePath(
	path string,
	openInfo os.FileInfo,
	expectedSize int64,
) error {
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf(
			"%w: inspect controlled executable %q: %w",
			ErrExecutableMaterializationFailed,
			path,
			err,
		)
	}
	if err := validateMaterializedExecutableInfo(path, pathInfo, expectedSize); err != nil {
		return err
	}
	if !os.SameFile(openInfo, pathInfo) {
		return fmt.Errorf(
			"%w: controlled path %q no longer identifies the written file",
			ErrMaterializedExecutableInvalid,
			path,
		)
	}

	return nil
}

func validateMaterializedExecutableInfo(
	path string,
	info os.FileInfo,
	expectedSize int64,
) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf(
			"%w: controlled executable %q is a symbolic link",
			ErrMaterializedExecutableInvalid,
			path,
		)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf(
			"%w: controlled executable %q is not a regular file",
			ErrMaterializedExecutableInvalid,
			path,
		)
	}
	if info.Size() != expectedSize {
		return fmt.Errorf(
			"%w: controlled executable %q has size %d, expected %d",
			ErrMaterializedExecutableInvalid,
			path,
			info.Size(),
			expectedSize,
		)
	}

	return nil
}
