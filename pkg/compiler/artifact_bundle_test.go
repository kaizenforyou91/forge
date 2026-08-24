package compiler

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

func TestNewArtifactBundle(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{
				Module: "http@v1",
			},
			{
				Module: "web@v1",
				Dependencies: []string{
					"http@v1",
				},
			},
		},
	}

	artifacts := []Artifact{
		{
			Module:     "http",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
		},
		{
			Module:     "web",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
		},
	}

	bundle, err := NewArtifactBundle(plan, artifacts)
	if err != nil {
		t.Fatal(err)
	}

	if bundle.ManifestName != "demo" {
		t.Fatalf(
			"expected manifest name %q, got %q",
			"demo",
			bundle.ManifestName,
		)
	}

	if bundle.ManifestVersion != "v1" {
		t.Fatalf(
			"expected manifest version %q, got %q",
			"v1",
			bundle.ManifestVersion,
		)
	}

	if !reflect.DeepEqual(bundle.Artifacts, artifacts) {
		t.Fatalf(
			"expected artifacts %#v, got %#v",
			artifacts,
			bundle.Artifacts,
		)
	}
}

func TestNewArtifactBundlePreservesArtifactOrder(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "logger@v1"},
			{Module: "http@v1"},
			{Module: "web@v1"},
		},
	}

	artifacts := []Artifact{
		{
			Module:     "logger",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/logger",
		},
		{
			Module:     "http",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
		},
		{
			Module:     "web",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
		},
	}

	bundle, err := NewArtifactBundle(plan, artifacts)
	if err != nil {
		t.Fatal(err)
	}

	for i, expected := range artifacts {
		if bundle.Artifacts[i] != expected {
			t.Fatalf(
				"artifact %d: expected %#v, got %#v",
				i,
				expected,
				bundle.Artifacts[i],
			)
		}
	}
}

func TestNewArtifactBundleRejectsArtifactCountMismatch(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "http@v1"},
		},
	}

	_, err := NewArtifactBundle(plan, nil)
	if err == nil {
		t.Fatal("expected count mismatch error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestNewArtifactBundleRejectsInvalidManifestIdentity(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "http@v1"},
		},
	}

	_, err := NewArtifactBundle(
		plan,
		[]Artifact{
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
		},
	)
	if err == nil {
		t.Fatal("expected invalid manifest identity error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestNewArtifactBundleRejectsArtifactIdentityMismatch(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "http@v1"},
		},
	}

	_, err := NewArtifactBundle(
		plan,
		[]Artifact{
			{
				Module:     "http",
				Version:    "v2",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
		},
	)
	if err == nil {
		t.Fatal("expected artifact identity mismatch")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}

	if !strings.Contains(err.Error(), "http@v2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewArtifactBundleRejectsInvalidArtifact(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "http@v1"},
		},
	}

	_, err := NewArtifactBundle(
		plan,
		[]Artifact{
			{Module: "", Version: "v1"},
		},
	)
	if err == nil {
		t.Fatal("expected invalid artifact error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestNewArtifactBundleRejectsMissingImportPath(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "http@v1"},
		},
	}

	_, err := NewArtifactBundle(
		plan,
		[]Artifact{
			{
				Module:  "http",
				Version: "v1",
			},
		},
	)

	if err == nil {
		t.Fatal("expected missing import path error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestNewArtifactBundleDoesNotAliasArtifacts(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{Module: "http@v1"},
			{Module: "web@v1"},
		},
	}

	artifacts := []Artifact{
		{
			Module:     "http",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
		},
		{
			Module:     "web",
			Version:    "v1",
			ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
		},
	}

	bundle, err := NewArtifactBundle(plan, artifacts)
	if err != nil {
		t.Fatal(err)
	}

	artifacts[0].Module = "changed"

	if bundle.Artifacts[0].Module != "http" {
		t.Fatal("bundle artifacts alias input slice")
	}
}

func TestArtifactBundleValidate(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
			{
				Module:     "web",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
			},
		},
	}

	if err := bundle.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactBundleValidateRejectsMissingManifestName(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestVersion: "v1",
	}

	err := bundle.Validate()
	if err == nil {
		t.Fatal("expected manifest name validation error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestArtifactBundleValidateRejectsMissingManifestVersion(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName: "demo",
	}

	err := bundle.Validate()
	if err == nil {
		t.Fatal("expected manifest version validation error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestArtifactBundleValidateRejectsInvalidArtifact(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{Module: "", Version: "v1"},
		},
	}

	err := bundle.Validate()
	if err == nil {
		t.Fatal("expected invalid artifact error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestArtifactBundleValidateRejectsMissingImportPath(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:  "http",
				Version: "v1",
			},
		},
	}

	err := bundle.Validate()
	if err == nil {
		t.Fatal("expected missing import path validation error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}
}

func TestArtifactBundleValidateRejectsDuplicateArtifact(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
		},
	}

	err := bundle.Validate()
	if err == nil {
		t.Fatal("expected duplicate artifact error")
	}

	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf(
			"expected ErrInvalidArtifactBundle, got %v",
			err,
		)
	}

	if !strings.Contains(err.Error(), "http@v1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArtifactBundleValidateAllowsEmptyArtifacts(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName:    "empty",
		ManifestVersion: "v1",
	}

	if err := bundle.Validate(); err != nil {
		t.Fatal(err)
	}
}

func testRunnableArtifactBundle() ArtifactBundle {
	return ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Runtime: &RuntimeDescriptor{
			Kind: RuntimeKindApplicationExecutable,
			Entrypoint: RuntimeEntrypoint{
				Module:  "demo",
				Version: "v1",
			},
			TargetOS:   "windows",
			TargetArch: "amd64",
		},
		Artifacts: []Artifact{{Module: "demo", Version: "v1", ImportPath: "example.com/demo"}},
	}
}

func TestArtifactBundleValidateForSchemaV1(t *testing.T) {
	if err := testArtifactBundle().ValidateForSchema(artifactBundleSchemaVersionV1); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactBundleV1RejectsRuntimeDescriptorForWriting(t *testing.T) {
	err := testRunnableArtifactBundle().ValidateForSchema(artifactBundleSchemaVersionV1)
	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf("expected ErrInvalidArtifactBundle, got %v", err)
	}
}

func TestArtifactBundleValidateForSchemaV2(t *testing.T) {
	if err := testRunnableArtifactBundle().ValidateForSchema(artifactBundleSchemaVersionV2); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactBundleV2RequiresRuntimeDescriptor(t *testing.T) {
	bundle := testArtifactBundle()
	err := bundle.ValidateForSchema(artifactBundleSchemaVersionV2)
	if !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf("expected ErrInvalidArtifactBundle, got %v", err)
	}
}

func TestArtifactBundleV2RejectsUnknownRuntimeKind(t *testing.T) {
	bundle := testRunnableArtifactBundle()
	bundle.Runtime.Kind = "unknown"
	requireInvalidArtifactBundle(t, bundle)
}

func TestArtifactBundleV2RejectsMissingTargetOS(t *testing.T) {
	bundle := testRunnableArtifactBundle()
	bundle.Runtime.TargetOS = " "
	requireInvalidArtifactBundle(t, bundle)
}

func TestArtifactBundleV2RejectsMissingTargetArch(t *testing.T) {
	bundle := testRunnableArtifactBundle()
	bundle.Runtime.TargetArch = " "
	requireInvalidArtifactBundle(t, bundle)
}

func TestArtifactBundleV2RejectsMissingEntrypointModule(t *testing.T) {
	bundle := testRunnableArtifactBundle()
	bundle.Runtime.Entrypoint.Module = " "
	requireInvalidArtifactBundle(t, bundle)
}

func TestArtifactBundleV2RejectsMissingEntrypointVersion(t *testing.T) {
	bundle := testRunnableArtifactBundle()
	bundle.Runtime.Entrypoint.Version = " "
	requireInvalidArtifactBundle(t, bundle)
}

func TestArtifactBundleV2RejectsUnmatchedEntrypoint(t *testing.T) {
	bundle := testRunnableArtifactBundle()
	bundle.Runtime.Entrypoint.Module = "missing"
	requireInvalidArtifactBundle(t, bundle)
}

func TestArtifactBundleV2AcceptsSingleMatchingEntrypoint(t *testing.T) {
	bundle := testRunnableArtifactBundle()
	bundle.Artifacts = append(bundle.Artifacts, Artifact{Module: "dependency", Version: "v1", ImportPath: "example.com/dependency"})
	if err := bundle.ValidateForSchema(artifactBundleSchemaVersionV2); err != nil {
		t.Fatal(err)
	}
}

func requireInvalidArtifactBundle(t *testing.T, bundle ArtifactBundle) {
	t.Helper()
	if err := bundle.ValidateForSchema(artifactBundleSchemaVersionV2); !errors.Is(err, ErrInvalidArtifactBundle) {
		t.Fatalf("expected ErrInvalidArtifactBundle, got %v", err)
	}
}
