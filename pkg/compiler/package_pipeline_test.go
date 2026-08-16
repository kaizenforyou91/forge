package compiler

import (
	"path/filepath"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

type fakeArtifactPackager struct {
	called     bool
	bundle     ArtifactBundle
	payloads   map[string][]byte
	outputPath string
}

func (p *fakeArtifactPackager) Package(
	bundle ArtifactBundle,
	payloads map[string][]byte,
	outputPath string,
) error {
	p.called = true
	p.bundle = bundle
	p.payloads = payloads
	p.outputPath = outputPath
	return nil
}

func TestPackageArtifactsBuildsBundleAndPayloads(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{
				Module: "http@v1",
			},
			{
				Module: "web@v1",
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

	packager := &fakeArtifactPackager{}
	output := filepath.Join(t.TempDir(), "demo.zip")

	if err := PackageArtifacts(
		plan,
		artifacts,
		packager,
		output,
	); err != nil {
		t.Fatal(err)
	}

	if !packager.called {
		t.Fatal("expected packager to be called")
	}

	if packager.outputPath != output {
		t.Fatalf(
			"expected output path %q, got %q",
			output,
			packager.outputPath,
		)
	}

	if packager.bundle.ManifestName != "demo" {
		t.Fatalf(
			"expected manifest name demo, got %q",
			packager.bundle.ManifestName,
		)
	}

	if len(packager.bundle.Artifacts) != 2 {
		t.Fatalf(
			"expected 2 artifacts, got %d",
			len(packager.bundle.Artifacts),
		)
	}

	if string(packager.payloads["http@v1"]) != "http@v1" {
		t.Fatal("unexpected http payload")
	}

	if string(packager.payloads["web@v1"]) != "web@v1" {
		t.Fatal("unexpected web payload")
	}
}

func TestPackageArtifactsRejectsNilPackager(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
	}

	err := PackageArtifacts(
		plan,
		nil,
		nil,
		filepath.Join(t.TempDir(), "demo.zip"),
	)

	if err == nil {
		t.Fatal("expected nil packager error")
	}
}

func TestPackageArtifactsRejectsArtifactMismatch(t *testing.T) {
	plan := manifest.BuildPlan{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Steps: []manifest.BuildStep{
			{
				Module: "http@v1",
			},
		},
	}

	artifacts := []Artifact{
		{
			Module:  "web",
			Version: "v1",
		},
	}

	packager := &fakeArtifactPackager{}

	err := PackageArtifacts(
		plan,
		artifacts,
		packager,
		filepath.Join(t.TempDir(), "demo.zip"),
	)

	if err == nil {
		t.Fatal("expected artifact mismatch error")
	}

	if packager.called {
		t.Fatal("packager must not run after bundle validation failure")
	}
}
