package compiler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	testRunnableApplicationImportPath = "github.com/kaizenforyou91/forge/pkg/compiler/testdata/runnable_app"
	testNonMainImportPath             = "github.com/kaizenforyou91/forge/pkg/compiler/testdata/non_main"
)

type executableBuilderFakeRunner struct {
	commands []Command
	contexts []context.Context
	run      func(int, context.Context, Command) (CommandResult, error)
}

type executableOutputFileInfo struct {
	mode os.FileMode
	size int64
}

func (i executableOutputFileInfo) Name() string       { return "application" }
func (i executableOutputFileInfo) Size() int64        { return i.size }
func (i executableOutputFileInfo) Mode() os.FileMode  { return i.mode }
func (i executableOutputFileInfo) ModTime() time.Time { return time.Time{} }
func (i executableOutputFileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i executableOutputFileInfo) Sys() any           { return nil }

func (r *executableBuilderFakeRunner) Run(
	ctx context.Context,
	command Command,
) (CommandResult, error) {
	index := len(r.commands)
	r.commands = append(r.commands, Command{
		Name: command.Name,
		Args: append([]string(nil), command.Args...),
		Dir:  command.Dir,
		Env:  append([]string(nil), command.Env...),
	})
	r.contexts = append(r.contexts, ctx)

	if r.run != nil {
		return r.run(index, ctx, command)
	}

	return CommandResult{}, nil
}

func testExecutableBuildRequest(
	t *testing.T,
) executableBuildRequest {
	t.Helper()

	workingDirectory := t.TempDir()
	outputDirectory := t.TempDir()

	return executableBuildRequest{
		Entrypoint: RuntimeEntrypoint{
			Module:  "demo",
			Version: "v1",
		},
		ImportPath:       "example.com/demo",
		WorkingDirectory: workingDirectory,
		OutputPath: filepath.Join(
			outputDirectory,
			testExecutableOutputName(),
		),
	}
}

func testExecutableOutputName() string {
	if runtime.GOOS == "windows" {
		return "application.exe"
	}

	return "application"
}

func successfulExecutableBuilderRunner(
	t *testing.T,
	request executableBuildRequest,
) *executableBuilderFakeRunner {
	t.Helper()

	return &executableBuilderFakeRunner{
		run: func(
			index int,
			_ context.Context,
			_ Command,
		) (CommandResult, error) {
			switch index {
			case 0:
				return CommandResult{Stdout: "main\n"}, nil
			case 1:
				if err := os.WriteFile(
					request.OutputPath,
					[]byte("fake-host-executable"),
					0o755,
				); err != nil {
					t.Fatal(err)
				}
				return CommandResult{}, nil
			default:
				t.Fatalf("unexpected command index %d", index)
				return CommandResult{}, nil
			}
		},
	}
}

func TestGoApplicationExecutableBuilderContract(t *testing.T) {
	var _ applicationExecutableBuilder = (*GoApplicationExecutableBuilder)(nil)
}

func TestNewGoApplicationExecutableBuilderRejectsNilRunner(t *testing.T) {
	_, err := NewGoApplicationExecutableBuilder(nil)
	if !errors.Is(err, ErrNilCommandRunner) {
		t.Fatalf("expected ErrNilCommandRunner, got %v", err)
	}
}

func TestGoApplicationExecutableBuilderBuildsHostExecutable(t *testing.T) {
	t.Setenv("GOCACHE", filepath.Join(t.TempDir(), "go-cache"))
	t.Setenv("GOTELEMETRY", "off")
	workingDirectory, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	outputDirectory := t.TempDir()
	outputPath := filepath.Join(outputDirectory, testExecutableOutputName())
	request := executableBuildRequest{
		Entrypoint: RuntimeEntrypoint{
			Module:  "runnable-app",
			Version: "v1",
		},
		ImportPath:       testRunnableApplicationImportPath,
		WorkingDirectory: workingDirectory,
		OutputPath:       outputPath,
	}
	builder, err := NewGoApplicationExecutableBuilder(NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}

	result, err := builder.Build(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}

	if result.Path != outputPath {
		t.Fatalf("expected path %q, got %q", outputPath, result.Path)
	}
	if result.TargetOS != runtime.GOOS {
		t.Fatalf("expected target OS %q, got %q", runtime.GOOS, result.TargetOS)
	}
	if result.TargetArch != runtime.GOARCH {
		t.Fatalf("expected target architecture %q, got %q", runtime.GOARCH, result.TargetArch)
	}
	if result.Entrypoint != request.Entrypoint {
		t.Fatalf("expected entrypoint %#v, got %#v", request.Entrypoint, result.Entrypoint)
	}
	if result.ImportPath != request.ImportPath {
		t.Fatalf("expected import path %q, got %q", request.ImportPath, result.ImportPath)
	}

	info, err := os.Lstat(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("expected regular executable output, got %v", info.Mode())
	}
	if info.Size() <= 0 {
		t.Fatal("expected non-empty executable output")
	}
	if runtime.GOOS == "windows" && filepath.Ext(outputPath) != ".exe" {
		t.Fatalf("expected Windows output to use .exe, got %q", outputPath)
	}
}

func TestGoApplicationExecutableBuilderUsesControlledOutputPath(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := successfulExecutableBuilderRunner(t, request)
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	result, err := builder.Build(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != request.OutputPath {
		t.Fatalf("expected controlled output %q, got %q", request.OutputPath, result.Path)
	}
	if _, err := os.Stat(request.OutputPath); err != nil {
		t.Fatal(err)
	}
}

func TestGoApplicationExecutableBuilderReportsHostTarget(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := successfulExecutableBuilderRunner(t, request)
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	result, err := builder.Build(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.TargetOS != runtime.GOOS || result.TargetArch != runtime.GOARCH {
		t.Fatalf(
			"expected host target %s/%s, got %s/%s",
			runtime.GOOS,
			runtime.GOARCH,
			result.TargetOS,
			result.TargetArch,
		)
	}
}

func TestGoApplicationExecutableBuilderRejectsInvalidImportPath(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := &executableBuilderFakeRunner{
		run: func(
			_ int,
			_ context.Context,
			_ Command,
		) (CommandResult, error) {
			return CommandResult{ExitCode: 1, Stderr: "package not found"}, nil
		},
	}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build(context.Background(), request)
	if !errors.Is(err, ErrExecutableBuildFailed) {
		t.Fatalf("expected ErrExecutableBuildFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "package not found") {
		t.Fatalf("expected stderr context, got %v", err)
	}
}

func TestGoApplicationExecutableBuilderRejectsNonMainPackage(t *testing.T) {
	t.Setenv("GOCACHE", filepath.Join(t.TempDir(), "go-cache"))
	t.Setenv("GOTELEMETRY", "off")
	workingDirectory, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	request := executableBuildRequest{
		Entrypoint:       RuntimeEntrypoint{Module: "library", Version: "v1"},
		ImportPath:       testNonMainImportPath,
		WorkingDirectory: workingDirectory,
		OutputPath:       filepath.Join(t.TempDir(), testExecutableOutputName()),
	}
	builder, err := NewGoApplicationExecutableBuilder(NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build(context.Background(), request)
	if !errors.Is(err, ErrInvalidApplicationEntrypoint) {
		t.Fatalf("expected ErrInvalidApplicationEntrypoint, got %v", err)
	}
	if _, statErr := os.Lstat(request.OutputPath); !os.IsNotExist(statErr) {
		t.Fatalf("non-main package must not produce output, got %v", statErr)
	}
}

func TestGoApplicationExecutableBuilderPropagatesCancellation(t *testing.T) {
	request := testExecutableBuildRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	runner := &executableBuilderFakeRunner{
		run: func(
			_ int,
			commandContext context.Context,
			_ Command,
		) (CommandResult, error) {
			cancel()
			return CommandResult{}, commandContext.Err()
		},
	}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	_, err = builder.Build(ctx, request)
	if !errors.Is(err, ErrExecutableBuildFailed) {
		t.Fatalf("expected ErrExecutableBuildFailed, got %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("expected cancellation during go list, got %d commands", len(runner.commands))
	}
}

func TestGoApplicationExecutableBuilderRejectsCanceledContext(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := &executableBuilderFakeRunner{}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = builder.Build(ctx, request)
	if !errors.Is(err, ErrExecutableBuildFailed) || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled executable build, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("expected no commands for canceled context, got %d", len(runner.commands))
	}
}

func TestGoApplicationExecutableBuilderRejectsMissingOutput(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := &executableBuilderFakeRunner{
		run: func(
			index int,
			_ context.Context,
			_ Command,
		) (CommandResult, error) {
			if index == 0 {
				return CommandResult{Stdout: "main"}, nil
			}
			return CommandResult{}, nil
		},
	}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build(context.Background(), request)
	if !errors.Is(err, ErrExecutableOutputMissing) {
		t.Fatalf("expected ErrExecutableOutputMissing, got %v", err)
	}
}

func TestGoApplicationExecutableBuilderRejectsEmptyOutput(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := &executableBuilderFakeRunner{
		run: func(
			index int,
			_ context.Context,
			_ Command,
		) (CommandResult, error) {
			if index == 0 {
				return CommandResult{Stdout: "main"}, nil
			}
			if err := os.WriteFile(request.OutputPath, nil, 0o755); err != nil {
				t.Fatal(err)
			}
			return CommandResult{}, nil
		},
	}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build(context.Background(), request)
	if !errors.Is(err, ErrExecutableOutputMissing) {
		t.Fatalf("expected ErrExecutableOutputMissing, got %v", err)
	}
}

func TestGoApplicationExecutableBuilderRejectsSymlinkOutput(t *testing.T) {
	err := validateExecutableOutputInfo(
		"application",
		executableOutputFileInfo{
			mode: os.ModeSymlink | 0o777,
			size: 1,
		},
	)
	if !errors.Is(err, ErrExecutableOutputMissing) {
		t.Fatalf("expected ErrExecutableOutputMissing, got %v", err)
	}
}

func TestGoApplicationExecutableBuilderRejectsNonRegularOutput(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := &executableBuilderFakeRunner{
		run: func(
			index int,
			_ context.Context,
			_ Command,
		) (CommandResult, error) {
			if index == 0 {
				return CommandResult{Stdout: "main"}, nil
			}
			if err := os.Mkdir(request.OutputPath, 0o755); err != nil {
				t.Fatal(err)
			}
			return CommandResult{}, nil
		},
	}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build(context.Background(), request)
	if !errors.Is(err, ErrExecutableOutputMissing) {
		t.Fatalf("expected ErrExecutableOutputMissing, got %v", err)
	}
}

func TestGoApplicationExecutableBuilderRejectsPreexistingOutput(t *testing.T) {
	request := testExecutableBuildRequest(t)
	if err := os.WriteFile(request.OutputPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := &executableBuilderFakeRunner{}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected pre-existing output rejection, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("expected no command execution, got %d", len(runner.commands))
	}
}

func TestGoApplicationExecutableBuilderRejectsInvalidWorkingDirectory(t *testing.T) {
	for _, test := range []struct {
		name             string
		workingDirectory func(*testing.T) string
	}{
		{
			name: "blank",
			workingDirectory: func(*testing.T) string {
				return " "
			},
		},
		{
			name: "missing",
			workingDirectory: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing")
			},
		},
		{
			name: "file",
			workingDirectory: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "file")
				if err := os.WriteFile(path, []byte("file"), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := testExecutableBuildRequest(t)
			request.WorkingDirectory = test.workingDirectory(t)
			runner := &executableBuilderFakeRunner{}
			builder, err := NewGoApplicationExecutableBuilder(runner)
			if err != nil {
				t.Fatal(err)
			}

			_, err = builder.Build(context.Background(), request)
			if err == nil {
				t.Fatal("expected invalid working directory error")
			}
			if len(runner.commands) != 0 {
				t.Fatalf("expected no command execution, got %d", len(runner.commands))
			}
		})
	}
}

func TestGoApplicationExecutableBuilderRejectsStructurallyInvalidRequest(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*executableBuildRequest)
	}{
		{
			name: "missing entrypoint module",
			mutate: func(request *executableBuildRequest) {
				request.Entrypoint.Module = " "
			},
		},
		{
			name: "missing entrypoint version",
			mutate: func(request *executableBuildRequest) {
				request.Entrypoint.Version = " "
			},
		},
		{
			name: "missing import path",
			mutate: func(request *executableBuildRequest) {
				request.ImportPath = " "
			},
		},
		{
			name: "missing output path",
			mutate: func(request *executableBuildRequest) {
				request.OutputPath = " "
			},
		},
		{
			name: "relative output path",
			mutate: func(request *executableBuildRequest) {
				request.OutputPath = testExecutableOutputName()
			},
		},
		{
			name: "missing output parent",
			mutate: func(request *executableBuildRequest) {
				request.OutputPath = filepath.Join(
					filepath.Dir(request.OutputPath),
					"missing",
					testExecutableOutputName(),
				)
			},
		},
		{
			name: "output parent is a file",
			mutate: func(request *executableBuildRequest) {
				parent := filepath.Join(filepath.Dir(request.OutputPath), "file-parent")
				if err := os.WriteFile(parent, []byte("file"), 0o644); err != nil {
					t.Fatal(err)
				}
				request.OutputPath = filepath.Join(parent, testExecutableOutputName())
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := testExecutableBuildRequest(t)
			test.mutate(&request)
			runner := &executableBuilderFakeRunner{}
			builder, err := NewGoApplicationExecutableBuilder(runner)
			if err != nil {
				t.Fatal(err)
			}

			_, err = builder.Build(context.Background(), request)
			if err == nil {
				t.Fatal("expected structural request error")
			}
			if len(runner.commands) != 0 {
				t.Fatalf("expected no command execution, got %d", len(runner.commands))
			}
		})
	}
}

func TestGoApplicationExecutableBuilderRejectsNilContext(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := &executableBuilderFakeRunner{}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build(nil, request)
	if err == nil || !strings.Contains(err.Error(), "context is nil") {
		t.Fatalf("expected nil context rejection, got %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("expected no command execution, got %d", len(runner.commands))
	}
}

func TestGoApplicationExecutableBuilderUsesHostGOOSGOARCH(t *testing.T) {
	t.Setenv("GoOs", "unsupported-os")
	t.Setenv("gOaRcH", "unsupported-arch")
	t.Setenv("FORGE_EXECUTABLE_BUILDER_PRESERVED", "preserved")
	request := testExecutableBuildRequest(t)
	runner := successfulExecutableBuilderRunner(t, request)
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := builder.Build(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected two commands, got %d", len(runner.commands))
	}

	for _, command := range runner.commands {
		values := make(map[string][]string)
		for _, entry := range command.Env {
			key, value, found := strings.Cut(entry, "=")
			if !found {
				continue
			}
			values[strings.ToUpper(key)] = append(values[strings.ToUpper(key)], value)
		}
		if !reflect.DeepEqual(values["GOOS"], []string{runtime.GOOS}) {
			t.Fatalf("expected one host GOOS, got %#v", values["GOOS"])
		}
		if !reflect.DeepEqual(values["GOARCH"], []string{runtime.GOARCH}) {
			t.Fatalf("expected one host GOARCH, got %#v", values["GOARCH"])
		}
		if !reflect.DeepEqual(
			values["FORGE_EXECUTABLE_BUILDER_PRESERVED"],
			[]string{"preserved"},
		) {
			t.Fatalf("expected unrelated environment preservation, got %#v", values)
		}
	}
}

func TestGoApplicationExecutableBuilderConstructsExactCommands(t *testing.T) {
	request := testExecutableBuildRequest(t)
	runner := successfulExecutableBuilderRunner(t, request)
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("key"), "value")

	if _, err := builder.Build(ctx, request); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("expected two commands, got %d", len(runner.commands))
	}

	wantListArgs := []string{
		"list",
		"-f={{.Name}}",
		request.ImportPath,
	}
	if runner.commands[0].Name != "go" ||
		!reflect.DeepEqual(runner.commands[0].Args, wantListArgs) {
		t.Fatalf("unexpected go list command %#v", runner.commands[0])
	}

	wantBuildArgs := []string{
		"build",
		"-trimpath",
		"-buildvcs=false",
		"-o",
		request.OutputPath,
		request.ImportPath,
	}
	if runner.commands[1].Name != "go" ||
		!reflect.DeepEqual(runner.commands[1].Args, wantBuildArgs) {
		t.Fatalf("unexpected go build command %#v", runner.commands[1])
	}

	for i, command := range runner.commands {
		if command.Dir != request.WorkingDirectory {
			t.Fatalf("command %d: expected directory %q, got %q", i, request.WorkingDirectory, command.Dir)
		}
		if command.Env == nil {
			t.Fatalf("command %d: expected controlled environment", i)
		}
		if runner.contexts[i] != ctx {
			t.Fatalf("command %d: caller context was not preserved", i)
		}
	}
	if !reflect.DeepEqual(runner.commands[0].Env, runner.commands[1].Env) {
		t.Fatal("expected go list and go build to use the same environment")
	}
}

func TestGoApplicationExecutableBuilderClassifiesBuildFailure(t *testing.T) {
	request := testExecutableBuildRequest(t)
	wantErr := errors.New("runner failed")
	runner := &executableBuilderFakeRunner{
		run: func(
			index int,
			_ context.Context,
			_ Command,
		) (CommandResult, error) {
			if index == 0 {
				return CommandResult{Stdout: "main"}, nil
			}
			return CommandResult{}, wantErr
		},
	}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}

	_, err = builder.Build(context.Background(), request)
	if !errors.Is(err, ErrExecutableBuildFailed) {
		t.Fatalf("expected ErrExecutableBuildFailed, got %v", err)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped runner error, got %v", err)
	}
}

func TestGoApplicationExecutableBuilderClassifiesCommandFailures(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(int) (CommandResult, error)
	}{
		{
			name: "go list runner failure",
			run: func(int) (CommandResult, error) {
				return CommandResult{}, errors.New("go list runner failure")
			},
		},
		{
			name: "go build non-zero exit",
			run: func(index int) (CommandResult, error) {
				if index == 0 {
					return CommandResult{Stdout: "main"}, nil
				}
				return CommandResult{ExitCode: 2, Stderr: "go build failure"}, nil
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := testExecutableBuildRequest(t)
			runner := &executableBuilderFakeRunner{
				run: func(
					index int,
					_ context.Context,
					_ Command,
				) (CommandResult, error) {
					return test.run(index)
				},
			}
			builder, err := NewGoApplicationExecutableBuilder(runner)
			if err != nil {
				t.Fatal(err)
			}

			_, err = builder.Build(context.Background(), request)
			if !errors.Is(err, ErrExecutableBuildFailed) {
				t.Fatalf("expected ErrExecutableBuildFailed, got %v", err)
			}
		})
	}
}
