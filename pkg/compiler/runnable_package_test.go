package compiler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

var fakeRunnableExecutableBytes = []byte("real-looking-test-binary-bytes")

type recordingApplicationExecutableBuilder struct {
	delegate applicationExecutableBuilder
	request  executableBuildRequest
	result   ExecutableBuildResult
	payload  []byte
}

func (b *recordingApplicationExecutableBuilder) Build(
	ctx context.Context,
	request executableBuildRequest,
) (ExecutableBuildResult, error) {
	b.request = request

	result, err := b.delegate.Build(ctx, request)
	if err != nil {
		return ExecutableBuildResult{}, err
	}

	payload, err := os.ReadFile(result.Path)
	if err != nil {
		return ExecutableBuildResult{}, err
	}

	b.result = result
	b.payload = append([]byte(nil), payload...)

	return result, nil
}

type realRunnablePackageResult struct {
	request       RunnablePackageRequest
	recorder      *recordingApplicationExecutableBuilder
	fixtureBefore map[string][]byte
	fixturePath   string
}

type fakeRunnablePackageSourceResolver struct {
	calls  []RuntimeEntrypoint
	source PackageSource
	err    error
}

func (r *fakeRunnablePackageSourceResolver) Resolve(
	name,
	version string,
) (PackageSource, error) {
	r.calls = append(r.calls, RuntimeEntrypoint{Module: name, Version: version})
	return r.source, r.err
}

type fakeRunnableExecutableBuilder struct {
	contexts    []context.Context
	requests    []executableBuildRequest
	err         error
	omitOutput  bool
	emptyOutput bool
	mutate      func(*ExecutableBuildResult)
}

func (b *fakeRunnableExecutableBuilder) Build(
	ctx context.Context,
	request executableBuildRequest,
) (ExecutableBuildResult, error) {
	b.contexts = append(b.contexts, ctx)
	b.requests = append(b.requests, request)

	if b.err != nil {
		return ExecutableBuildResult{}, b.err
	}

	if !b.omitOutput {
		payload := fakeRunnableExecutableBytes
		if b.emptyOutput {
			payload = []byte{}
		}
		if err := os.WriteFile(request.OutputPath, payload, 0o755); err != nil {
			return ExecutableBuildResult{}, err
		}
	}

	result := ExecutableBuildResult{
		Path:       request.OutputPath,
		TargetOS:   runtime.GOOS,
		TargetArch: runtime.GOARCH,
		Entrypoint: request.Entrypoint,
		ImportPath: request.ImportPath,
	}
	if b.mutate != nil {
		b.mutate(&result)
	}

	return result, nil
}

type fakeRunnablePackagePackager struct {
	bundles     []ArtifactBundle
	payloads    []map[string][]byte
	outputPaths []string
	err         error
}

func (p *fakeRunnablePackagePackager) packageRunnable(
	bundle ArtifactBundle,
	payloads map[string][]byte,
	outputPath string,
) error {
	p.bundles = append(p.bundles, cloneRunnablePackageBundle(bundle))
	p.payloads = append(p.payloads, clonePayloads(payloads))
	p.outputPaths = append(p.outputPaths, outputPath)
	return p.err
}

func cloneRunnablePackageBundle(bundle ArtifactBundle) ArtifactBundle {
	result := bundle
	result.Artifacts = append([]Artifact(nil), bundle.Artifacts...)
	if bundle.Runtime != nil {
		runtimeCopy := *bundle.Runtime
		result.Runtime = &runtimeCopy
	}
	return result
}

func runnablePackageTestPlan() manifest.BuildPlan {
	return manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "core@v1"},
			{
				Module:       "demo@v1",
				Dependencies: []string{"core@v1"},
			},
		},
	}
}

func cloneRunnablePackagePlan(plan manifest.BuildPlan) manifest.BuildPlan {
	result := plan
	result.Steps = make([]manifest.BuildStep, len(plan.Steps))
	for i, step := range plan.Steps {
		result.Steps[i] = manifest.BuildStep{
			Module:       step.Module,
			Dependencies: append([]string(nil), step.Dependencies...),
		}
	}
	return result
}

func runnablePackageTestRequest(t *testing.T) RunnablePackageRequest {
	t.Helper()
	return RunnablePackageRequest{
		Plan: runnablePackageTestPlan(),
		Entrypoint: RuntimeEntrypoint{
			Module:  "demo",
			Version: "v1",
		},
		WorkingDirectory: t.TempDir(),
		OutputPath:       filepath.Join(t.TempDir(), "demo-v1.zip"),
	}
}

func newRunnablePackageTestCompiler(
	t *testing.T,
	builder *fakeRunnableExecutableBuilder,
	packager runnablePackagePackager,
) (*RunnablePackageCompiler, *fakeRunnablePackageSourceResolver) {
	t.Helper()

	resolver := &fakeRunnablePackageSourceResolver{
		source: PackageSource{
			Name:       "demo",
			Version:    "v1",
			ImportPath: "example.com/demo",
		},
	}
	compiler, err := newRunnablePackageCompiler(resolver, builder, packager)
	if err != nil {
		t.Fatal(err)
	}

	return compiler, resolver
}

func requireRunnableTemporaryDirectoryRemoved(
	t *testing.T,
	builder *fakeRunnableExecutableBuilder,
) {
	t.Helper()
	if len(builder.requests) != 1 {
		t.Fatalf("expected one builder request, got %d", len(builder.requests))
	}
	temporaryDirectory := filepath.Dir(builder.requests[0].OutputPath)
	if _, err := os.Stat(temporaryDirectory); !os.IsNotExist(err) {
		t.Fatalf("expected temporary directory %q to be removed, got %v", temporaryDirectory, err)
	}
}

func TestNewRunnablePackageCompilerRejectsNilDependencies(t *testing.T) {
	runner := &fakeCommandRunner{}
	builder, err := NewGoApplicationExecutableBuilder(runner)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &fakeRunnablePackageSourceResolver{}

	if _, err := NewRunnablePackageCompiler(nil, builder, NewZIPPackager()); !errors.Is(err, ErrInvalidPackageSource) {
		t.Fatalf("expected ErrInvalidPackageSource, got %v", err)
	}
	if _, err := NewRunnablePackageCompiler(resolver, nil, NewZIPPackager()); !errors.Is(err, ErrExecutableBuildFailed) {
		t.Fatalf("expected ErrExecutableBuildFailed, got %v", err)
	}
	if _, err := NewRunnablePackageCompiler(resolver, builder, nil); !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf("expected ErrInvalidArtifactPackage, got %v", err)
	}
}

func TestRunnablePackageCompilerRequiresEntrypoint(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*RunnablePackageRequest)
	}{
		{
			name: "module",
			mutate: func(request *RunnablePackageRequest) {
				request.Entrypoint.Module = " "
			},
		},
		{
			name: "version",
			mutate: func(request *RunnablePackageRequest) {
				request.Entrypoint.Version = " "
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := runnablePackageTestRequest(t)
			test.mutate(&request)
			builder := &fakeRunnableExecutableBuilder{}
			packager := &fakeRunnablePackagePackager{}
			compiler, resolver := newRunnablePackageTestCompiler(t, builder, packager)

			err := compiler.Compile(context.Background(), request)
			if !errors.Is(err, ErrInvalidApplicationEntrypoint) {
				t.Fatalf("expected ErrInvalidApplicationEntrypoint, got %v", err)
			}
			if len(resolver.calls) != 0 || len(builder.requests) != 0 {
				t.Fatal("invalid request must fail before source resolution or build")
			}
		})
	}
}

func TestRunnablePackageCompilerRejectsInvalidRequestBeforeMutation(t *testing.T) {
	workingFile := filepath.Join(t.TempDir(), "working-file")
	if err := os.WriteFile(workingFile, []byte("not-a-directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name    string
		context context.Context
		mutate  func(*RunnablePackageRequest)
	}{
		{
			name:    "nil context",
			context: nil,
			mutate:  func(*RunnablePackageRequest) {},
		},
		{
			name:    "missing working directory",
			context: context.Background(),
			mutate: func(request *RunnablePackageRequest) {
				request.WorkingDirectory = " "
			},
		},
		{
			name:    "working directory does not exist",
			context: context.Background(),
			mutate: func(request *RunnablePackageRequest) {
				request.WorkingDirectory = filepath.Join(t.TempDir(), "missing")
			},
		},
		{
			name:    "working directory is a file",
			context: context.Background(),
			mutate: func(request *RunnablePackageRequest) {
				request.WorkingDirectory = workingFile
			},
		},
		{
			name:    "missing output path",
			context: context.Background(),
			mutate: func(request *RunnablePackageRequest) {
				request.OutputPath = " "
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := runnablePackageTestRequest(t)
			test.mutate(&request)
			builder := &fakeRunnableExecutableBuilder{}
			packager := &fakeRunnablePackagePackager{}
			compiler, resolver := newRunnablePackageTestCompiler(t, builder, packager)

			err := compiler.Compile(test.context, request)
			if err == nil {
				t.Fatal("expected invalid request error")
			}
			if len(resolver.calls) != 0 || len(builder.requests) != 0 || len(packager.bundles) != 0 {
				t.Fatal("invalid request must fail before resolution, build, or packaging")
			}
		})
	}
}

func TestRunnablePackageCompilerRejectsCanceledContextBeforeMutation(t *testing.T) {
	request := runnablePackageTestRequest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	builder := &fakeRunnableExecutableBuilder{}
	packager := &fakeRunnablePackagePackager{}
	compiler, resolver := newRunnablePackageTestCompiler(t, builder, packager)

	err := compiler.Compile(ctx, request)
	if !errors.Is(err, ErrExecutableBuildFailed) || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled executable build failure, got %v", err)
	}
	if len(resolver.calls) != 0 || len(builder.requests) != 0 || len(packager.bundles) != 0 {
		t.Fatal("canceled request must fail before resolution, build, or packaging")
	}
}

func TestRunnablePackageCompilerRejectsEntrypointOutsidePlan(t *testing.T) {
	request := runnablePackageTestRequest(t)
	request.Entrypoint = RuntimeEntrypoint{Module: "missing", Version: "v1"}
	builder := &fakeRunnableExecutableBuilder{}
	compiler, resolver := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	err := compiler.Compile(context.Background(), request)
	if !errors.Is(err, ErrInvalidApplicationEntrypoint) {
		t.Fatalf("expected ErrInvalidApplicationEntrypoint, got %v", err)
	}
	if len(resolver.calls) != 0 || len(builder.requests) != 0 {
		t.Fatal("missing entrypoint must fail before source resolution or build")
	}
}

func TestRunnablePackageCompilerRejectsAmbiguousEntrypoint(t *testing.T) {
	request := runnablePackageTestRequest(t)
	request.Plan.Steps = append(
		request.Plan.Steps,
		manifest.BuildStep{Module: "demo@v1"},
	)
	builder := &fakeRunnableExecutableBuilder{}
	compiler, _ := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	err := compiler.Compile(context.Background(), request)
	if !errors.Is(err, ErrInvalidApplicationEntrypoint) {
		t.Fatalf("expected ErrInvalidApplicationEntrypoint, got %v", err)
	}
}

func TestRunnablePackageCompilerRejectsInvalidBuildPlan(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*manifest.BuildPlan)
	}{
		{
			name: "manifest identity",
			mutate: func(plan *manifest.BuildPlan) {
				plan.ManifestName = ""
			},
		},
		{
			name: "step identity",
			mutate: func(plan *manifest.BuildPlan) {
				plan.Steps[0].Module = "invalid"
			},
		},
		{
			name: "dependency identity",
			mutate: func(plan *manifest.BuildPlan) {
				plan.Steps[1].Dependencies[0] = "invalid"
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := runnablePackageTestRequest(t)
			test.mutate(&request.Plan)
			builder := &fakeRunnableExecutableBuilder{}
			compiler, _ := newRunnablePackageTestCompiler(
				t, builder, &fakeRunnablePackagePackager{},
			)

			err := compiler.Compile(context.Background(), request)
			if !errors.Is(err, ErrInvalidBuildPlan) {
				t.Fatalf("expected ErrInvalidBuildPlan, got %v", err)
			}
			if len(builder.requests) != 0 {
				t.Fatal("invalid plan must fail before build")
			}
		})
	}
}

func TestRunnablePackageCompilerResolvesEntrypointImportPath(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	compiler, resolver := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	wantCall := []RuntimeEntrypoint{request.Entrypoint}
	if !reflect.DeepEqual(resolver.calls, wantCall) {
		t.Fatalf("expected source calls %#v, got %#v", wantCall, resolver.calls)
	}
	if builder.requests[0].ImportPath != resolver.source.ImportPath {
		t.Fatalf("expected resolved import path %q, got %q", resolver.source.ImportPath, builder.requests[0].ImportPath)
	}
}

func TestRunnablePackageCompilerRejectsInvalidResolvedSource(t *testing.T) {
	for _, source := range []PackageSource{
		{Name: "other", Version: "v1", ImportPath: "example.com/demo"},
		{Name: "demo", Version: "other", ImportPath: "example.com/demo"},
		{Name: "demo", Version: "v1", ImportPath: " "},
	} {
		request := runnablePackageTestRequest(t)
		builder := &fakeRunnableExecutableBuilder{}
		resolver := &fakeRunnablePackageSourceResolver{source: source}
		compiler, err := newRunnablePackageCompiler(
			resolver, builder, &fakeRunnablePackagePackager{},
		)
		if err != nil {
			t.Fatal(err)
		}

		err = compiler.Compile(context.Background(), request)
		if !errors.Is(err, ErrInvalidApplicationEntrypoint) {
			t.Fatalf("source %#v: expected ErrInvalidApplicationEntrypoint, got %v", source, err)
		}
		if len(builder.requests) != 0 {
			t.Fatal("invalid resolved source must fail before build")
		}
	}
}

func TestRunnablePackageCompilerWrapsEntrypointSourceResolutionFailure(t *testing.T) {
	request := runnablePackageTestRequest(t)
	wantErr := errors.New("entrypoint source unavailable")
	resolver := &fakeRunnablePackageSourceResolver{err: wantErr}
	builder := &fakeRunnableExecutableBuilder{}
	compiler, err := newRunnablePackageCompiler(
		resolver,
		builder,
		&fakeRunnablePackagePackager{},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = compiler.Compile(context.Background(), request)
	if !errors.Is(err, ErrInvalidApplicationEntrypoint) || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped entrypoint source failure, got %v", err)
	}
	if len(builder.requests) != 0 {
		t.Fatal("source resolution failure must occur before build")
	}
}

func TestRunnablePackageCompilerPassesControlledBuilderRequest(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	compiler, resolver := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)
	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("key"), "value")

	if err := compiler.Compile(ctx, request); err != nil {
		t.Fatal(err)
	}
	if len(builder.requests) != 1 || len(builder.contexts) != 1 {
		t.Fatalf("expected one builder call, got %d", len(builder.requests))
	}
	got := builder.requests[0]
	if got.Entrypoint != request.Entrypoint ||
		got.ImportPath != resolver.source.ImportPath ||
		got.WorkingDirectory != request.WorkingDirectory {
		t.Fatalf("unexpected builder request %#v", got)
	}
	if builder.contexts[0] != ctx {
		t.Fatal("caller context was not preserved")
	}
}

func TestRunnablePackageCompilerUsesPrivateTemporaryExecutablePath(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	compiler, _ := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	got := builder.requests[0].OutputPath
	if !filepath.IsAbs(got) {
		t.Fatalf("expected absolute private output path, got %q", got)
	}
	if !strings.HasPrefix(filepath.Base(filepath.Dir(got)), "forge-runnable-build-") {
		t.Fatalf("expected private runnable build directory, got %q", filepath.Dir(got))
	}
	if filepath.Dir(got) == filepath.Dir(request.OutputPath) {
		t.Fatal("temporary executable must not share the final package directory")
	}
	requireRunnableTemporaryDirectoryRemoved(t, builder)
}

func TestRunnablePackageCompilerUsesWindowsExecutableSuffix(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows executable suffix contract")
	}
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	compiler, _ := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(builder.requests[0].OutputPath) != "application.exe" {
		t.Fatalf("expected application.exe, got %q", builder.requests[0].OutputPath)
	}
}

func TestRunnablePackageCompilerRejectsBuilderResultMismatch(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*ExecutableBuildResult)
	}{
		{
			name: "path",
			mutate: func(result *ExecutableBuildResult) {
				result.Path += ".other"
			},
		},
		{
			name: "entrypoint",
			mutate: func(result *ExecutableBuildResult) {
				result.Entrypoint.Module = "other"
			},
		},
		{
			name: "import path",
			mutate: func(result *ExecutableBuildResult) {
				result.ImportPath = "example.com/other"
			},
		},
		{
			name: "empty target OS",
			mutate: func(result *ExecutableBuildResult) {
				result.TargetOS = ""
			},
		},
		{
			name: "empty target architecture",
			mutate: func(result *ExecutableBuildResult) {
				result.TargetArch = ""
			},
		},
		{
			name: "target mismatch",
			mutate: func(result *ExecutableBuildResult) {
				result.TargetOS = "unsupported-host"
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := runnablePackageTestRequest(t)
			builder := &fakeRunnableExecutableBuilder{mutate: test.mutate}
			compiler, _ := newRunnablePackageTestCompiler(
				t, builder, &fakeRunnablePackagePackager{},
			)

			err := compiler.Compile(context.Background(), request)
			if !errors.Is(err, ErrExecutableBuildFailed) {
				t.Fatalf("expected ErrExecutableBuildFailed, got %v", err)
			}
			requireRunnableTemporaryDirectoryRemoved(t, builder)
		})
	}
}

func TestRunnablePackageCompilerRejectsMissingBuilderOutput(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{omitOutput: true}
	compiler, _ := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	err := compiler.Compile(context.Background(), request)
	if !errors.Is(err, ErrExecutableOutputMissing) {
		t.Fatalf("expected ErrExecutableOutputMissing, got %v", err)
	}
	requireRunnableTemporaryDirectoryRemoved(t, builder)
}

func TestRunnablePackageCompilerRejectsEmptyBuilderOutput(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{emptyOutput: true}
	compiler, _ := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	err := compiler.Compile(context.Background(), request)
	if !errors.Is(err, ErrExecutableOutputMissing) {
		t.Fatalf("expected ErrExecutableOutputMissing, got %v", err)
	}
	requireRunnableTemporaryDirectoryRemoved(t, builder)
}

func TestRunnablePackageCompilerBuildsSingleArtifactBundleV2(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	packager := &fakeRunnablePackagePackager{}
	compiler, resolver := newRunnablePackageTestCompiler(t, builder, packager)

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(packager.bundles) != 1 {
		t.Fatalf("expected one package, got %d", len(packager.bundles))
	}
	if len(packager.outputPaths) != 1 || packager.outputPaths[0] != request.OutputPath {
		t.Fatalf("expected final output path %q, got %#v", request.OutputPath, packager.outputPaths)
	}
	bundle := packager.bundles[0]
	if err := bundle.ValidateForSchema(artifactBundleSchemaVersionV2); err != nil {
		t.Fatal(err)
	}
	if bundle.ManifestName != request.Plan.ManifestName ||
		bundle.ManifestVersion != request.Plan.ManifestVersion {
		t.Fatalf("unexpected manifest identity %#v", bundle)
	}
	if bundle.Runtime == nil ||
		bundle.Runtime.Kind != RuntimeKindApplicationExecutable ||
		bundle.Runtime.Entrypoint != request.Entrypoint ||
		bundle.Runtime.TargetOS != runtime.GOOS ||
		bundle.Runtime.TargetArch != runtime.GOARCH {
		t.Fatalf("unexpected runtime descriptor %#v", bundle.Runtime)
	}
	if len(bundle.Artifacts) != 1 {
		t.Fatalf("expected one executable artifact, got %d", len(bundle.Artifacts))
	}
	wantArtifact := Artifact{
		Module:     request.Entrypoint.Module,
		Version:    request.Entrypoint.Version,
		ImportPath: resolver.source.ImportPath,
	}
	if bundle.Artifacts[0] != wantArtifact {
		t.Fatalf("expected artifact %#v, got %#v", wantArtifact, bundle.Artifacts[0])
	}
}

func TestRunnablePackageCompilerPackagesExactBuilderOutput(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	packager := &fakeRunnablePackagePackager{}
	compiler, _ := newRunnablePackageTestCompiler(t, builder, packager)

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(packager.payloads) != 1 || len(packager.payloads[0]) != 1 {
		t.Fatalf("expected one payload, got %#v", packager.payloads)
	}
	if !bytes.Equal(packager.payloads[0]["demo@v1"], fakeRunnableExecutableBytes) {
		t.Fatalf("expected exact builder bytes, got %q", packager.payloads[0]["demo@v1"])
	}
	if _, exists := packager.payloads[0]["core@v1"]; exists {
		t.Fatal("dependency identity payload must not be packaged")
	}
}

func TestRunnablePackageCompilerCreatesUnsignedV2Package(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	resolver := &fakeRunnablePackageSourceResolver{
		source: PackageSource{Name: "demo", Version: "v1", ImportPath: "example.com/demo"},
	}
	compiler, err := newRunnablePackageCompiler(
		resolver,
		builder,
		&zipRunnablePackagePackager{packager: NewZIPPackager()},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	entries := readZIPEntriesForTest(t, request.OutputPath)
	wantMetadata := []byte(`{"package_format_version":2,"bundle_schema_version":2}`)
	if !bytes.Equal(entries[packageMetadataPath], wantMetadata) {
		t.Fatalf("expected metadata %s, got %s", wantMetadata, entries[packageMetadataPath])
	}
	if _, exists := entries[signatureManifestPath]; exists {
		t.Fatal("unsigned runnable package must not contain signature.json")
	}

	assertRunnablePackageReadBack(t, NewZIPPackageReader(), request)
	requireRunnableTemporaryDirectoryRemoved(t, builder)
}

func TestRunnablePackageCompilerCreatesSignedV2Package(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	signer, verifier := trustedTestSignerAndVerifier(t)
	resolver := &fakeRunnablePackageSourceResolver{
		source: PackageSource{Name: "demo", Version: "v1", ImportPath: "example.com/demo"},
	}
	compiler, err := newRunnablePackageCompiler(
		resolver,
		builder,
		&zipRunnablePackagePackager{
			packager: NewZIPPackagerWithSigner(signer),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	entries := readZIPEntriesForTest(t, request.OutputPath)
	signature, err := UnmarshalPackageSignature(entries[signatureManifestPath])
	if err != nil {
		t.Fatal(err)
	}
	if signature.Version != packageSignatureVersion {
		t.Fatalf("expected signature schema %d, got %d", packageSignatureVersion, signature.Version)
	}

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertRunnablePackageReadBack(t, reader, request)
	requireRunnableTemporaryDirectoryRemoved(t, builder)
}

func TestRunnablePackageCompilerSignedPackageAuthenticatesExecutablePayload(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	signer, verifier := trustedTestSignerAndVerifier(t)
	resolver := &fakeRunnablePackageSourceResolver{
		source: PackageSource{Name: "demo", Version: "v1", ImportPath: "example.com/demo"},
	}
	compiler, err := newRunnablePackageCompiler(
		resolver,
		builder,
		&zipRunnablePackagePackager{
			packager: NewZIPPackagerWithSigner(signer),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	entries := readZIPEntriesForTest(t, request.OutputPath)
	entries["artifacts/demo/v1/artifact"] = []byte("tampered-test-binary-bytes")
	tamperedPath := filepath.Join(t.TempDir(), "tampered-runnable.zip")
	writeZIPEntriesForTest(t, tamperedPath, entries)

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reader.Read(tamperedPath); !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch for executable payload tamper, got %v", err)
	}
}

func assertRunnablePackageReadBack(
	t *testing.T,
	reader *ZIPPackageReader,
	request RunnablePackageRequest,
) {
	t.Helper()
	bundle, payloads, err := reader.Read(request.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Runtime == nil ||
		bundle.Runtime.Kind != RuntimeKindApplicationExecutable ||
		bundle.Runtime.Entrypoint != request.Entrypoint ||
		bundle.Runtime.TargetOS != runtime.GOOS ||
		bundle.Runtime.TargetArch != runtime.GOARCH {
		t.Fatalf("unexpected read-back runtime %#v", bundle.Runtime)
	}
	if len(bundle.Artifacts) != 1 {
		t.Fatalf("expected one read-back artifact, got %d", len(bundle.Artifacts))
	}
	artifact := bundle.Artifacts[0]
	if artifact.Module != request.Entrypoint.Module ||
		artifact.Version != request.Entrypoint.Version ||
		artifact.ImportPath != "example.com/demo" {
		t.Fatalf("unexpected read-back artifact %#v", artifact)
	}
	if len(payloads) != 1 ||
		!bytes.Equal(payloads["demo@v1"], fakeRunnableExecutableBytes) {
		t.Fatalf("unexpected read-back payloads %#v", payloads)
	}
}

func TestRunnablePackageCompilerCleansTemporaryOutputOnSuccess(t *testing.T) {
	request := runnablePackageTestRequest(t)
	builder := &fakeRunnableExecutableBuilder{}
	compiler, _ := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	requireRunnableTemporaryDirectoryRemoved(t, builder)
}

func TestRunnablePackageCompilerCleansTemporaryOutputOnBuildFailure(t *testing.T) {
	request := runnablePackageTestRequest(t)
	wantErr := errors.New("fake executable build failed")
	builder := &fakeRunnableExecutableBuilder{err: wantErr}
	compiler, _ := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	err := compiler.Compile(context.Background(), request)
	if !errors.Is(err, ErrExecutableBuildFailed) || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped build failure, got %v", err)
	}
	requireRunnableTemporaryDirectoryRemoved(t, builder)
}

func TestRunnablePackageCompilerCleansTemporaryOutputOnPackagingFailure(t *testing.T) {
	request := runnablePackageTestRequest(t)
	wantErr := errors.New("fake runnable packaging failed")
	builder := &fakeRunnableExecutableBuilder{}
	compiler, _ := newRunnablePackageTestCompiler(
		t,
		builder,
		&fakeRunnablePackagePackager{err: wantErr},
	)

	err := compiler.Compile(context.Background(), request)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected packaging failure, got %v", err)
	}
	requireRunnableTemporaryDirectoryRemoved(t, builder)
}

func TestRunnablePackageCompilerDoesNotMutatePlan(t *testing.T) {
	request := runnablePackageTestRequest(t)
	want := cloneRunnablePackagePlan(request.Plan)
	builder := &fakeRunnableExecutableBuilder{}
	compiler, _ := newRunnablePackageTestCompiler(
		t, builder, &fakeRunnablePackagePackager{},
	)

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(request.Plan, want) {
		t.Fatalf("runnable compiler mutated plan\nwant %#v\ngot  %#v", want, request.Plan)
	}
}

func TestRunnablePackageCompilerPackagesRealHostExecutable(t *testing.T) {
	result := compileRealRunnablePackage(t, NewZIPPackager())
	payload := result.recorder.payload

	if len(payload) == 0 {
		t.Fatal("expected non-empty real executable bytes")
	}
	if bytes.Equal(payload, []byte("placeholder-not-executable")) {
		t.Fatal("real executable must not equal the package-v2 placeholder")
	}
	if bytes.Equal(payload, []byte("runnable-app@v1")) {
		t.Fatal("real executable must not equal a v1 identity payload")
	}
	if result.recorder.result.TargetOS != runtime.GOOS ||
		result.recorder.result.TargetArch != runtime.GOARCH {
		t.Fatalf(
			"expected host target %s/%s, got %s/%s",
			runtime.GOOS,
			runtime.GOARCH,
			result.recorder.result.TargetOS,
			result.recorder.result.TargetArch,
		)
	}

	if _, err := os.Lstat(result.recorder.result.Path); !os.IsNotExist(err) {
		t.Fatalf("expected temporary executable to be removed, got %v", err)
	}
	if _, err := os.Lstat(filepath.Dir(result.recorder.result.Path)); !os.IsNotExist(err) {
		t.Fatalf("expected temporary build directory to be removed, got %v", err)
	}
	if _, err := os.Stat(result.request.OutputPath); err != nil {
		t.Fatalf("expected final package to remain: %v", err)
	}

	entries := readZIPEntriesForTest(t, result.request.OutputPath)
	wantMetadata := []byte(`{"package_format_version":2,"bundle_schema_version":2}`)
	if !bytes.Equal(entries[packageMetadataPath], wantMetadata) {
		t.Fatalf("expected package-v2 metadata %s, got %s", wantMetadata, entries[packageMetadataPath])
	}

	integrity, err := UnmarshalPackageIntegrity(entries[integrityManifestPath])
	if err != nil {
		t.Fatal(err)
	}
	if integrity.Version != packageIntegrityVersion {
		t.Fatalf("expected integrity version %d, got %d", packageIntegrityVersion, integrity.Version)
	}
	if len(integrity.Artifacts) != 1 {
		t.Fatalf("expected one integrity artifact, got %d", len(integrity.Artifacts))
	}
	digest := sha256.Sum256(payload)
	wantDigest := hex.EncodeToString(digest[:])
	if integrity.Artifacts[0].SHA256 != wantDigest {
		t.Fatalf("expected executable digest %q, got %q", wantDigest, integrity.Artifacts[0].SHA256)
	}

	bundle, payloads, err := NewZIPPackageReader().Read(result.request.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	assertRealRunnablePackageReadBack(t, bundle, payloads, result.request, payload)

	payloadPath := "artifacts/runnable-app/v1/artifact"
	tamperedEntries := readZIPEntriesForTest(t, result.request.OutputPath)
	tamperedEntries[payloadPath] = mutateRealExecutablePayload(t, tamperedEntries[payloadPath])
	tamperedPath := filepath.Join(t.TempDir(), "tampered-real-runnable.zip")
	writeZIPEntriesForTest(t, tamperedPath, tamperedEntries)
	if _, _, err := NewZIPPackageReader().Read(tamperedPath); !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch for real executable tamper, got %v", err)
	}

	fixtureAfter := snapshotRunnableApplicationFixture(t, result.fixturePath)
	if !reflect.DeepEqual(fixtureAfter, result.fixtureBefore) {
		t.Fatalf("real build mutated source fixture\nbefore %#v\nafter  %#v", result.fixtureBefore, fixtureAfter)
	}
}

func TestRunnablePackageCompilerCreatesStrictlyVerifiableSignedRealExecutablePackage(t *testing.T) {
	signer, verifier := trustedTestSignerAndVerifier(t)
	result := compileRealRunnablePackage(t, NewZIPPackagerWithSigner(signer))

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	bundle, payloads, err := reader.Read(result.request.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	assertRealRunnablePackageReadBack(
		t,
		bundle,
		payloads,
		result.request,
		result.recorder.payload,
	)

	entries := readZIPEntriesForTest(t, result.request.OutputPath)
	signature, err := UnmarshalPackageSignature(entries[signatureManifestPath])
	if err != nil {
		t.Fatal(err)
	}
	if signature.Version != packageSignatureVersion {
		t.Fatalf("expected signature version %d, got %d", packageSignatureVersion, signature.Version)
	}

	payloadPath := "artifacts/runnable-app/v1/artifact"
	payloadTamper := readZIPEntriesForTest(t, result.request.OutputPath)
	payloadTamper[payloadPath] = mutateRealExecutablePayload(t, payloadTamper[payloadPath])
	payloadTamperPath := filepath.Join(t.TempDir(), "signed-payload-tamper.zip")
	writeZIPEntriesForTest(t, payloadTamperPath, payloadTamper)
	if _, _, err := reader.Read(payloadTamperPath); !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch for signed real payload tamper, got %v", err)
	}

	runtimeTamper := readZIPEntriesForTest(t, result.request.OutputPath)
	oldTarget := []byte(`"target_arch":"` + runtime.GOARCH + `"`)
	newTarget := []byte(`"target_arch":"tampered-` + runtime.GOARCH + `"`)
	if bytes.Count(runtimeTamper[bundleManifestPath], oldTarget) != 1 {
		t.Fatalf("expected exactly one canonical target_arch field in %s", runtimeTamper[bundleManifestPath])
	}
	runtimeTamper[bundleManifestPath] = bytes.Replace(
		runtimeTamper[bundleManifestPath],
		oldTarget,
		newTarget,
		1,
	)
	runtimeTamperPath := filepath.Join(t.TempDir(), "signed-runtime-tamper.zip")
	writeZIPEntriesForTest(t, runtimeTamperPath, runtimeTamper)
	if _, _, err := reader.Read(runtimeTamperPath); !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch for signed runtime metadata tamper, got %v", err)
	}
}

func compileRealRunnablePackage(
	t *testing.T,
	packager *ZIPPackager,
) realRunnablePackageResult {
	t.Helper()
	t.Setenv("GOTELEMETRY", "off")

	workingDirectory, fixturePath := runnablePackageRepositoryPaths(t)
	fixtureBefore := snapshotRunnableApplicationFixture(t, fixturePath)
	entrypoint := RuntimeEntrypoint{Module: "runnable-app", Version: "v1"}
	request := RunnablePackageRequest{
		Plan: manifest.BuildPlan{
			ManifestName:    "real-runnable",
			ManifestVersion: "v1",
			Steps: []manifest.BuildStep{
				{Module: "runnable-app@v1"},
			},
		},
		Entrypoint:       entrypoint,
		WorkingDirectory: workingDirectory,
		OutputPath:       filepath.Join(t.TempDir(), "real-runnable-v2.zip"),
	}

	sources := NewPackageSourceRegistry()
	if err := sources.Register(PackageSource{
		Name:       entrypoint.Module,
		Version:    entrypoint.Version,
		ImportPath: testRunnableApplicationImportPath,
	}); err != nil {
		t.Fatal(err)
	}
	realBuilder, err := NewGoApplicationExecutableBuilder(NewOSCommandRunner())
	if err != nil {
		t.Fatal(err)
	}
	recorder := &recordingApplicationExecutableBuilder{delegate: realBuilder}
	compiler, err := newRunnablePackageCompiler(
		sources,
		recorder,
		&zipRunnablePackagePackager{packager: packager},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := compiler.Compile(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	return realRunnablePackageResult{
		request:       request,
		recorder:      recorder,
		fixtureBefore: fixtureBefore,
		fixturePath:   fixturePath,
	}
}

func runnablePackageRepositoryPaths(t *testing.T) (string, string) {
	t.Helper()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve runnable package test source path")
	}
	compilerDirectory := filepath.Dir(testFile)
	repositoryRoot := filepath.Clean(filepath.Join(compilerDirectory, "..", ".."))
	if _, err := os.Stat(filepath.Join(repositoryRoot, "go.mod")); err != nil {
		t.Fatalf("resolve repository root %q: %v", repositoryRoot, err)
	}

	return repositoryRoot, filepath.Join(compilerDirectory, "testdata", "runnable_app")
}

func snapshotRunnableApplicationFixture(
	t *testing.T,
	directory string,
) map[string][]byte {
	t.Helper()

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			snapshot[entry.Name()+string(filepath.Separator)] = nil
			continue
		}

		data, err := os.ReadFile(filepath.Join(directory, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		snapshot[entry.Name()] = data
	}

	return snapshot
}

func assertRealRunnablePackageReadBack(
	t *testing.T,
	bundle ArtifactBundle,
	payloads map[string][]byte,
	request RunnablePackageRequest,
	wantPayload []byte,
) {
	t.Helper()

	if bundle.Runtime == nil ||
		bundle.Runtime.Kind != RuntimeKindApplicationExecutable ||
		bundle.Runtime.Entrypoint != request.Entrypoint ||
		bundle.Runtime.TargetOS != runtime.GOOS ||
		bundle.Runtime.TargetArch != runtime.GOARCH {
		t.Fatalf("unexpected real runnable descriptor %#v", bundle.Runtime)
	}
	if len(bundle.Artifacts) != 1 {
		t.Fatalf("expected one real executable artifact, got %d", len(bundle.Artifacts))
	}
	wantArtifact := Artifact{
		Module:     "runnable-app",
		Version:    "v1",
		ImportPath: testRunnableApplicationImportPath,
	}
	if bundle.Artifacts[0] != wantArtifact {
		t.Fatalf("expected real artifact %#v, got %#v", wantArtifact, bundle.Artifacts[0])
	}
	if len(payloads) != 1 || !bytes.Equal(payloads["runnable-app@v1"], wantPayload) {
		t.Fatal("package payload does not equal exact real builder output")
	}
}

func mutateRealExecutablePayload(t *testing.T, payload []byte) []byte {
	t.Helper()
	if len(payload) == 0 {
		t.Fatal("cannot mutate empty executable payload")
	}

	result := append([]byte(nil), payload...)
	result[len(result)/2] ^= 0xff
	return result
}
