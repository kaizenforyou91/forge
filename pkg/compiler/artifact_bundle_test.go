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
			Module:  "http",
			Version: "v1",
		},
		{
			Module:  "web",
			Version: "v1",
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
		{Module: "logger", Version: "v1"},
		{Module: "http", Version: "v1"},
		{Module: "web", Version: "v1"},
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
			{Module: "http", Version: "v1"},
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
			{Module: "http", Version: "v2"},
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
		{Module: "http", Version: "v1"},
		{Module: "web", Version: "v1"},
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
			{Module: "http", Version: "v1"},
			{Module: "web", Version: "v1"},
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

func TestArtifactBundleValidateRejectsDuplicateArtifact(t *testing.T) {
	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{Module: "http", Version: "v1"},
			{Module: "http", Version: "v1"},
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
