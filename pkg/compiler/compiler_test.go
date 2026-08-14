package compiler

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

type recordingCompiler struct {
	called bool
	plan   manifest.BuildPlan
}

func (c *recordingCompiler) Compile(
	plan manifest.BuildPlan,
) ([]Artifact, error) {
	c.called = true
	c.plan = plan

	artifacts := make([]Artifact, 0, len(plan.Steps))

	for _, step := range plan.Steps {
		name, version, ok := splitModuleKey(step.Module)
		if !ok {
			return nil, ErrInvalidBuildPlan
		}

		artifacts = append(artifacts, Artifact{
			Module:  name,
			Version: version,
		})
	}

	return artifacts, nil
}

func splitModuleKey(key string) (string, string, bool) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] != '@' {
			continue
		}

		if i == 0 || i == len(key)-1 {
			return "", "", false
		}

		return key[:i], key[i+1:], true
	}

	return "", "", false
}

func validBuildPlan() manifest.BuildPlan {
	return manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "demo",
		Steps: []manifest.BuildStep{
			{
				Module: "logger@v1",
			},
			{
				Module: "http@v1",
				Dependencies: []string{
					"logger@v1",
				},
			},
			{
				Module: "web@v1",
				Dependencies: []string{
					"http@v1",
					"logger@v1",
				},
			},
		},
	}
}

func TestCompilerContract(t *testing.T) {
	var _ Compiler = (*recordingCompiler)(nil)
}

func TestCompilerReceivesBuildPlan(t *testing.T) {
	compiler := &recordingCompiler{}
	plan := validBuildPlan()

	artifacts, err := compiler.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if !compiler.called {
		t.Fatal("expected compiler to be called")
	}

	if !reflect.DeepEqual(compiler.plan, plan) {
		t.Fatalf(
			"compiler received different build plan:\nwant %#v\ngot  %#v",
			plan,
			compiler.plan,
		)
	}

	if len(artifacts) != len(plan.Steps) {
		t.Fatalf(
			"expected %d artifacts, got %d",
			len(plan.Steps),
			len(artifacts),
		)
	}
}

func TestCompilerProducesDeterministicArtifactOrder(t *testing.T) {
	compiler := &recordingCompiler{}

	plan := validBuildPlan()

	first, err := compiler.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	second, err := compiler.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf(
			"compiler output is not deterministic:\nfirst %#v\nsecond %#v",
			first,
			second,
		)
	}

	want := []Artifact{
		{
			Module:  "logger",
			Version: "v1",
		},
		{
			Module:  "http",
			Version: "v1",
		},
		{
			Module:  "web",
			Version: "v1",
		},
	}

	if !reflect.DeepEqual(first, want) {
		t.Fatalf(
			"unexpected artifact order:\nwant %#v\ngot  %#v",
			want,
			first,
		)
	}
}

func TestCompilerPreservesPlanDependencySemantics(t *testing.T) {
	compiler := &recordingCompiler{}

	plan := validBuildPlan()

	_, err := compiler.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(
		compiler.plan.Steps[2].Dependencies,
		[]string{
			"http@v1",
			"logger@v1",
		},
	) {
		t.Fatalf(
			"compiler did not preserve dependency semantics: %#v",
			compiler.plan.Steps[2].Dependencies,
		)
	}
}

func TestCompilerRejectsMalformedModuleIdentity(t *testing.T) {
	compiler := &recordingCompiler{}

	plan := manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "invalid",
		Steps: []manifest.BuildStep{
			{
				Module: "invalid-module",
			},
		},
	}

	_, err := compiler.Compile(plan)
	if !errors.Is(err, ErrInvalidBuildPlan) {
		t.Fatalf(
			"expected ErrInvalidBuildPlan, got %v",
			err,
		)
	}
}

func TestCompilerAllowsEmptyBuildPlan(t *testing.T) {
	compiler := &recordingCompiler{}

	plan := manifest.BuildPlan{
		ManifestVersion: "v1",
		ManifestName:    "empty",
	}

	artifacts, err := compiler.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if len(artifacts) != 0 {
		t.Fatalf(
			"expected 0 artifacts, got %d",
			len(artifacts),
		)
	}
}

func TestCompilerDoesNotMutateBuildPlan(t *testing.T) {
	compiler := &recordingCompiler{}

	plan := validBuildPlan()
	original := plan

	_, err := compiler.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(plan, original) {
		t.Fatal("compiler mutated build plan")
	}
}
