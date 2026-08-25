package runtime

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kaizenforyou91/forge/pkg/compiler"
)

var materializerTestBytes = []byte("verified-materializer-test-bytes")

func TestSecureExecutableMaterializerMaterializesVerifiedBytes(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)

	data, err := os.ReadFile(materialized.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, materializerTestBytes) {
		t.Fatalf("materialized bytes = %q, want %q", data, materializerTestBytes)
	}
}

func TestSecureExecutableMaterializerUsesPrivateDirectory(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)

	if !filepath.IsAbs(materialized.directory) {
		t.Fatalf("private directory %q is not absolute", materialized.directory)
	}
	if !strings.HasPrefix(filepath.Base(materialized.directory), "forge-runtime-") {
		t.Fatalf("unexpected private directory %q", materialized.directory)
	}
	info, err := os.Lstat(materialized.directory)
	if err != nil {
		t.Fatal(err)
	}
	if err := validatePrivateMaterializationDirectory(materialized.directory, info); err != nil {
		t.Fatal(err)
	}
}

func TestSecureExecutableMaterializerUsesControlledFilename(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)

	expected := "application"
	if goruntime.GOOS == "windows" {
		expected = "application.exe"
	}
	if name := filepath.Base(materialized.path); name != expected {
		t.Fatalf("materialized filename = %q, want %q", name, expected)
	}
}

func TestSecureExecutableMaterializerWritesExactBytes(t *testing.T) {
	want := []byte{0x00, 0x01, 0x7f, 0x80, 0xfe, 0xff}
	materialized := materializeTestPackage(t, want)

	got, err := os.ReadFile(materialized.path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("materialized bytes = %v, want %v", got, want)
	}
}

func TestSecureExecutableMaterializerValidatesSHA256(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)

	want := sha256.Sum256(materializerTestBytes)
	if materialized.expectedSHA256 != want {
		t.Fatalf("stored SHA-256 = %x, want %x", materialized.expectedSHA256, want)
	}
	if materialized.expectedSize != int64(len(materializerTestBytes)) {
		t.Fatalf("stored size = %d, want %d", materialized.expectedSize, len(materializerTestBytes))
	}
}

func TestSecureExecutableMaterializerRejectsZeroVerifiedPackage(t *testing.T) {
	materializer := NewSecureExecutableMaterializer()
	result, err := materializer.Materialize(VerifiedRunnablePackage{})
	if !errors.Is(err, ErrInvalidRunnablePackage) {
		t.Fatalf("expected ErrInvalidRunnablePackage, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil materialized result")
	}

	var nilMaterializer *SecureExecutableMaterializer
	if _, err := nilMaterializer.Materialize(validMaterializerTestPackage(materializerTestBytes)); !errors.Is(err, ErrInvalidRunnablePackage) {
		t.Fatalf("expected ErrInvalidRunnablePackage from nil materializer, got %v", err)
	}
}

func TestSecureExecutableMaterializerRejectsIncompleteVerifiedPackage(t *testing.T) {
	tests := map[string]func(*VerifiedRunnablePackage){
		"package format": func(pkg *VerifiedRunnablePackage) { pkg.packageFormatVersion = 0 },
		"bundle schema":  func(pkg *VerifiedRunnablePackage) { pkg.bundleSchemaVersion = 0 },
		"manifest name":  func(pkg *VerifiedRunnablePackage) { pkg.manifestName = " " },
		"manifest version": func(pkg *VerifiedRunnablePackage) {
			pkg.manifestVersion = " "
		},
		"entrypoint module": func(pkg *VerifiedRunnablePackage) { pkg.entrypoint.Module = " " },
		"entrypoint version": func(pkg *VerifiedRunnablePackage) {
			pkg.entrypoint.Version = " "
		},
		"import path": func(pkg *VerifiedRunnablePackage) { pkg.importPath = " " },
		"signer":      func(pkg *VerifiedRunnablePackage) { pkg.signerKeyID = " " },
		"target OS":   func(pkg *VerifiedRunnablePackage) { pkg.targetOS = " " },
		"target arch": func(pkg *VerifiedRunnablePackage) { pkg.targetArch = " " },
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			pkg := validMaterializerTestPackage(materializerTestBytes)
			mutate(&pkg)

			_, err := NewSecureExecutableMaterializer().Materialize(pkg)
			if !errors.Is(err, ErrInvalidRunnablePackage) {
				t.Fatalf("expected ErrInvalidRunnablePackage, got %v", err)
			}
		})
	}
}

func TestSecureExecutableMaterializerRejectsWrongHost(t *testing.T) {
	pkg := validMaterializerTestPackage(materializerTestBytes)
	pkg.targetOS = "forge-unsupported-os"

	result, err := NewSecureExecutableMaterializer().Materialize(pkg)
	if !errors.Is(err, ErrUnsupportedRuntimePlatform) {
		t.Fatalf("expected ErrUnsupportedRuntimePlatform, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil materialized result")
	}
}

func TestSecureExecutableMaterializerRejectsEmptyExecutable(t *testing.T) {
	pkg := validMaterializerTestPackage(nil)

	result, err := NewSecureExecutableMaterializer().Materialize(pkg)
	if !errors.Is(err, ErrInvalidRunnablePackage) {
		t.Fatalf("expected ErrInvalidRunnablePackage, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil materialized result")
	}
}

func TestSecureExecutableMaterializerCreatesRegularFile(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)

	info, err := os.Lstat(materialized.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		t.Fatalf("unexpected materialized mode %v", info.Mode())
	}
	if info.Size() != int64(len(materializerTestBytes)) {
		t.Fatalf("materialized size = %d, want %d", info.Size(), len(materializerTestBytes))
	}
}

func TestSecureExecutableMaterializerUsesOwnerOnlyExecutableMode(t *testing.T) {
	if goruntime.GOOS == "windows" {
		t.Skip("Windows file mode bits do not represent executable ACLs")
	}

	materialized := materializeTestPackage(t, materializerTestBytes)
	info, err := os.Lstat(materialized.path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("materialized mode = %04o, want 0700", got)
	}
}

func TestMaterializedExecutableCloseRemovesDirectory(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	directory := materialized.directory

	if err := materialized.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		t.Fatalf("private directory still exists or cannot be inspected: %v", err)
	}
}

func TestMaterializedExecutableCloseIsIdempotent(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)

	if err := materialized.Close(); err != nil {
		t.Fatal(err)
	}
	if err := materialized.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}

	var nilMaterialized *MaterializedExecutable
	if err := nilMaterialized.Close(); err != nil {
		t.Fatalf("nil Close failed: %v", err)
	}
}

func TestMaterializedExecutableCloseIsConcurrencySafe(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	var cleanupCalls atomic.Int32
	materialized.removeAll = func(path string) error {
		cleanupCalls.Add(1)
		return os.RemoveAll(path)
	}

	const callers = 32
	errorsChannel := make(chan error, callers)
	var waitGroup sync.WaitGroup
	waitGroup.Add(callers)
	for range callers {
		go func() {
			defer waitGroup.Done()
			errorsChannel <- materialized.Close()
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)

	for err := range errorsChannel {
		if err != nil {
			t.Fatalf("concurrent Close failed: %v", err)
		}
	}
	if got := cleanupCalls.Load(); got != 1 {
		t.Fatalf("cleanup calls = %d, want 1", got)
	}
}

func TestMaterializedExecutableCleanupCanRetryAfterFailure(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "materialized")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}

	cleanupFailure := errors.New("injected cleanup failure")
	calls := 0
	materialized := &MaterializedExecutable{
		directory: directory,
		path:      filepath.Join(directory, "application"),
		removeAll: func(path string) error {
			calls++
			if calls == 1 {
				return cleanupFailure
			}
			return os.RemoveAll(path)
		},
	}

	err := materialized.Close()
	if !errors.Is(err, ErrExecutableMaterializationFailed) ||
		!errors.Is(err, cleanupFailure) {
		t.Fatalf("expected materialization and cleanup errors, got %v", err)
	}
	if !materialized.closed || materialized.cleanupDone {
		t.Fatalf("unexpected lifecycle after failed cleanup: closed=%v cleanupDone=%v", materialized.closed, materialized.cleanupDone)
	}
	if materialized.directory != directory {
		t.Fatal("failed cleanup discarded retry directory")
	}

	if err := materialized.Close(); err != nil {
		t.Fatalf("cleanup retry failed: %v", err)
	}
	if !materialized.closed || !materialized.cleanupDone {
		t.Fatalf("unexpected lifecycle after retry: closed=%v cleanupDone=%v", materialized.closed, materialized.cleanupDone)
	}
	if calls != 2 {
		t.Fatalf("cleanup calls = %d, want 2", calls)
	}
}

func TestSecureExecutableMaterializerCreatesIndependentMaterializations(t *testing.T) {
	materializer := NewSecureExecutableMaterializer()
	first, err := materializer.Materialize(validMaterializerTestPackage([]byte("first")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close() })
	second, err := materializer.Materialize(validMaterializerTestPackage([]byte("second")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })

	if first.directory == second.directory || first.path == second.path {
		t.Fatalf("materializations collided: first=%q second=%q", first.path, second.path)
	}
	firstBytes, err := os.ReadFile(first.path)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(second.path)
	if err != nil {
		t.Fatal(err)
	}
	if string(firstBytes) != "first" || string(secondBytes) != "second" {
		t.Fatalf("unexpected independent bytes: first=%q second=%q", firstBytes, secondBytes)
	}
}

func TestMaterializedExecutableMetadataSurvivesClose(t *testing.T) {
	materialized := materializeTestPackage(t, materializerTestBytes)
	wantEntrypoint := compiler.RuntimeEntrypoint{Module: "materializer-app", Version: "v1"}

	if err := materialized.Close(); err != nil {
		t.Fatal(err)
	}
	if materialized.Entrypoint() != wantEntrypoint {
		t.Fatalf("entrypoint = %#v, want %#v", materialized.Entrypoint(), wantEntrypoint)
	}
	if materialized.SignerKeyID() != "materializer-test-signer" {
		t.Fatalf("signer KeyID = %q", materialized.SignerKeyID())
	}
	if materialized.TargetOS() != goruntime.GOOS || materialized.TargetArch() != goruntime.GOARCH {
		t.Fatalf("target = %s/%s", materialized.TargetOS(), materialized.TargetArch())
	}
}

func TestSecureExecutableMaterializerWriteAllHandlesPartialWrites(t *testing.T) {
	writer := &limitedMaterializerWriter{limit: 2}
	data := []byte("partial-write")
	if err := writeAll(writer, data); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(writer.buffer.Bytes(), data) {
		t.Fatalf("written bytes = %q, want %q", writer.buffer.Bytes(), data)
	}

	if err := writeAll(zeroMaterializerWriter{}, []byte("data")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("expected io.ErrShortWrite, got %v", err)
	}
}

func TestSecureExecutableMaterializerRejectsInvalidFileInfo(t *testing.T) {
	tests := map[string]materializerFileInfo{
		"symlink": {
			mode: os.ModeSymlink | 0o700,
			size: 4,
		},
		"non-regular": {
			mode: os.ModeDir | 0o700,
			size: 4,
		},
		"size mismatch": {
			mode: 0o700,
			size: 3,
		},
	}

	for name, info := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateMaterializedExecutableInfo("application", info, 4)
			if !errors.Is(err, ErrMaterializedExecutableInvalid) {
				t.Fatalf("expected ErrMaterializedExecutableInvalid, got %v", err)
			}
		})
	}
}

func TestSecureExecutableMaterializerRejectsDigestMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "application")
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	data := []byte("actual")
	if err := writeAll(file, data); err != nil {
		t.Fatal(err)
	}
	wrongDigest := sha256.Sum256([]byte("different"))
	_, err = validateOpenMaterializedExecutable(file, path, int64(len(data)), wrongDigest)
	if !errors.Is(err, ErrMaterializedExecutableInvalid) {
		t.Fatalf("expected ErrMaterializedExecutableInvalid, got %v", err)
	}
}

func TestSecureExecutableMaterializerRejectsFilePathIdentityMismatch(t *testing.T) {
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "first")
	secondPath := filepath.Join(directory, "second")
	for _, path := range []string{firstPath, secondPath} {
		if err := os.WriteFile(path, []byte("same"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	firstInfo, err := os.Lstat(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	err = validateMaterializedExecutablePath(secondPath, firstInfo, 4)
	if !errors.Is(err, ErrMaterializedExecutableInvalid) {
		t.Fatalf("expected ErrMaterializedExecutableInvalid, got %v", err)
	}
}

func materializeTestPackage(t *testing.T, executable []byte) *MaterializedExecutable {
	t.Helper()

	materialized, err := NewSecureExecutableMaterializer().Materialize(
		validMaterializerTestPackage(executable),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := materialized.Close(); err != nil {
			t.Errorf("cleanup materialized executable: %v", err)
		}
	})

	return materialized
}

func validMaterializerTestPackage(executable []byte) VerifiedRunnablePackage {
	return VerifiedRunnablePackage{
		packageFormatVersion: runnablePackageFormatVersion,
		bundleSchemaVersion:  runnableBundleSchemaVersion,
		manifestName:         "materializer-fixture",
		manifestVersion:      "v1",
		entrypoint: compiler.RuntimeEntrypoint{
			Module:  "materializer-app",
			Version: "v1",
		},
		importPath:  "example.com/materializer-app",
		targetOS:    goruntime.GOOS,
		targetArch:  goruntime.GOARCH,
		signerKeyID: "materializer-test-signer",
		executable:  append([]byte(nil), executable...),
	}
}

type limitedMaterializerWriter struct {
	buffer bytes.Buffer
	limit  int
}

func (w *limitedMaterializerWriter) Write(data []byte) (int, error) {
	if len(data) > w.limit {
		data = data[:w.limit]
	}
	return w.buffer.Write(data)
}

type zeroMaterializerWriter struct{}

func (zeroMaterializerWriter) Write([]byte) (int, error) {
	return 0, nil
}

type materializerFileInfo struct {
	mode os.FileMode
	size int64
}

func (info materializerFileInfo) Name() string       { return "application" }
func (info materializerFileInfo) Size() int64        { return info.size }
func (info materializerFileInfo) Mode() os.FileMode  { return info.mode }
func (info materializerFileInfo) ModTime() time.Time { return time.Time{} }
func (info materializerFileInfo) IsDir() bool        { return info.mode.IsDir() }
func (info materializerFileInfo) Sys() any           { return nil }
