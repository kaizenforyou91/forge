package runtime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"
)

const (
	runtimeProcessWorkDirectoryName = "work"
	runtimeProcessOutputLimit       = 1 * 1024 * 1024
	runtimeProcessWaitDelay         = 2 * time.Second
)

// ProcessRunner starts one materialized executable directly through its
// package-private execution lease. It accepts no path, arguments, environment,
// working-directory, or shell input.
type ProcessRunner struct{}

func NewProcessRunner() *ProcessRunner {
	return &ProcessRunner{}
}

// Start validates and directly starts one single-use materialized executable.
// One background waiter owns child reaping and lease release.
func (r *ProcessRunner) Start(
	ctx context.Context,
	executable *MaterializedExecutable,
) (*RunningProcess, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: process runner is nil", ErrProcessStartFailed)
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is nil", ErrProcessStartFailed)
	}
	if executable == nil {
		return nil, fmt.Errorf(
			"%w: materialized executable is nil",
			ErrMaterializedExecutableInvalid,
		)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	lease, err := executable.acquireExecutionLease()
	if err != nil {
		return nil, err
	}

	if lease.targetOS != goruntime.GOOS || lease.targetArch != goruntime.GOARCH {
		return nil, releaseLeaseAfterStartFailure(
			lease,
			fmt.Errorf(
				"%w: lease targets %s/%s, host is %s/%s",
				ErrMaterializedExecutableInvalid,
				lease.targetOS,
				lease.targetArch,
				goruntime.GOOS,
				goruntime.GOARCH,
			),
		)
	}

	workDirectory, err := createRuntimeProcessWorkDirectory(lease.directory)
	if err != nil {
		return nil, releaseLeaseAfterStartFailure(lease, err)
	}
	if err := validateExecutableForStart(lease); err != nil {
		return nil, releaseLeaseAfterStartFailure(lease, err)
	}

	stdout := newBoundedOutputWriter(runtimeProcessOutputLimit)
	stderr := newBoundedOutputWriter(runtimeProcessOutputLimit)
	cmd := exec.CommandContext(ctx, lease.path)
	cmd.Dir = workDirectory
	cmd.Env = runtimeProcessEnvironment(workDirectory)
	cmd.Stdin = nil
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = runtimeProcessWaitDelay

	if err := cmd.Start(); err != nil {
		return nil, releaseLeaseAfterStartFailure(
			lease,
			fmt.Errorf(
				"%w: start direct child: %w",
				ErrProcessStartFailed,
				err,
			),
		)
	}

	process := newRunningProcess(ctx, cmd, lease, stdout, stderr)
	go process.waitInBackground()
	return process, nil
}

func newRunningProcess(
	ctx context.Context,
	cmd *exec.Cmd,
	lease *executableLease,
	stdout,
	stderr *boundedOutputWriter,
) *RunningProcess {
	return &RunningProcess{
		pid:         cmd.Process.Pid,
		entrypoint:  lease.entrypoint,
		signerKeyID: lease.signerKeyID,
		cmd:         cmd,
		ctx:         ctx,
		lease:       lease,
		stdout:      stdout,
		stderr:      stderr,
		done:        make(chan struct{}),
	}
}

func releaseLeaseAfterStartFailure(lease *executableLease, primary error) error {
	if releaseErr := lease.release(); releaseErr != nil {
		return errors.Join(primary, releaseErr)
	}
	return primary
}

func createRuntimeProcessWorkDirectory(directory string) (string, error) {
	workDirectory := filepath.Join(directory, runtimeProcessWorkDirectoryName)
	if err := os.Mkdir(workDirectory, 0o700); err != nil {
		return "", fmt.Errorf(
			"%w: create controlled working directory %q: %w",
			ErrProcessStartFailed,
			workDirectory,
			err,
		)
	}

	info, err := os.Lstat(workDirectory)
	if err != nil {
		return "", fmt.Errorf(
			"%w: inspect controlled working directory %q: %w",
			ErrProcessStartFailed,
			workDirectory,
			err,
		)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf(
			"%w: controlled working path %q is not a real directory",
			ErrProcessStartFailed,
			workDirectory,
		)
	}
	if goruntime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf(
			"%w: controlled working directory %q grants group or other permissions %04o",
			ErrProcessStartFailed,
			workDirectory,
			info.Mode().Perm(),
		)
	}

	return workDirectory, nil
}

func runtimeProcessEnvironment(workDirectory string) []string {
	if goruntime.GOOS == "windows" {
		environment := []string{
			"USERPROFILE=" + workDirectory,
			"TEMP=" + workDirectory,
			"TMP=" + workDirectory,
		}
		if systemRoot := inheritedEnvironmentValue("SYSTEMROOT"); systemRoot != "" {
			environment = append(environment, "SYSTEMROOT="+systemRoot)
		}
		return environment
	}

	return []string{
		"HOME=" + workDirectory,
		"TMPDIR=" + workDirectory,
	}
}

func inheritedEnvironmentValue(name string) string {
	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, name) {
			return value
		}
	}
	return ""
}

func validateExecutableForStart(lease *executableLease) (resultErr error) {
	file, err := os.Open(lease.path)
	if err != nil {
		return fmt.Errorf(
			"%w: open controlled executable %q: %w",
			ErrMaterializedExecutableInvalid,
			lease.path,
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
				"%w: close validation handle %q: %w",
				ErrMaterializedExecutableInvalid,
				lease.path,
				closeErr,
			)
			if resultErr == nil {
				resultErr = wrapped
			} else {
				resultErr = errors.Join(resultErr, wrapped)
			}
		}
	}()

	openInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf(
			"%w: stat validation handle %q: %w",
			ErrMaterializedExecutableInvalid,
			lease.path,
			err,
		)
	}
	if err := validateMaterializedExecutableInfo(
		lease.path,
		openInfo,
		lease.expectedSize,
	); err != nil {
		return err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf(
			"%w: seek validation handle %q: %w",
			ErrMaterializedExecutableInvalid,
			lease.path,
			err,
		)
	}
	hasher := sha256.New()
	read, err := io.Copy(hasher, file)
	if err != nil {
		return fmt.Errorf(
			"%w: hash controlled executable %q: %w",
			ErrMaterializedExecutableInvalid,
			lease.path,
			err,
		)
	}
	if read != lease.expectedSize {
		return fmt.Errorf(
			"%w: controlled executable %q read %d bytes, expected %d",
			ErrMaterializedExecutableInvalid,
			lease.path,
			read,
			lease.expectedSize,
		)
	}
	var actualSHA256 [32]byte
	copy(actualSHA256[:], hasher.Sum(nil))
	if actualSHA256 != lease.expectedSHA256 {
		return fmt.Errorf(
			"%w: controlled executable %q SHA-256 changed before start",
			ErrMaterializedExecutableInvalid,
			lease.path,
		)
	}

	if err := validateExecutableHeader(file); err != nil {
		return err
	}
	if err := validateStartExecutablePath(
		lease.path,
		openInfo,
		lease.expectedSize,
	); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		fileOpen = false
		return fmt.Errorf(
			"%w: close validation handle %q: %w",
			ErrMaterializedExecutableInvalid,
			lease.path,
			err,
		)
	}
	fileOpen = false

	return validateStartExecutablePath(
		lease.path,
		openInfo,
		lease.expectedSize,
	)
}

func validateStartExecutablePath(
	path string,
	openInfo os.FileInfo,
	expectedSize int64,
) error {
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf(
			"%w: inspect controlled executable path %q: %w",
			ErrMaterializedExecutableInvalid,
			path,
			err,
		)
	}
	if err := validateMaterializedExecutableInfo(path, pathInfo, expectedSize); err != nil {
		return err
	}
	if !os.SameFile(openInfo, pathInfo) {
		return fmt.Errorf(
			"%w: controlled executable path %q changed identity before start",
			ErrMaterializedExecutableInvalid,
			path,
		)
	}
	return nil
}

type boundedOutputWriter struct {
	mu        sync.Mutex
	limit     int
	buffer    bytes.Buffer
	truncated bool
}

func newBoundedOutputWriter(limit int) *boundedOutputWriter {
	return &boundedOutputWriter{limit: limit}
}

func (w *boundedOutputWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	accepted := len(data)
	remaining := w.limit - w.buffer.Len()
	if remaining > 0 {
		retained := len(data)
		if retained > remaining {
			retained = remaining
		}
		_, _ = w.buffer.Write(data[:retained])
	}
	if accepted > remaining {
		w.truncated = true
	}
	return accepted, nil
}

func (w *boundedOutputWriter) snapshot() ([]byte, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]byte(nil), w.buffer.Bytes()...), w.truncated
}
