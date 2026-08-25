package manifest

import (
	"errors"
	"reflect"
	"testing"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

func TestBuildPlanForManifest(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	plan, err := BuildPlanForManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	if plan.ManifestVersion != "v1" {
		t.Fatalf(
			"expected manifest version %q, got %q",
			"v1",
			plan.ManifestVersion,
		)
	}

	if plan.ManifestName != "demo" {
		t.Fatalf(
			"expected manifest name %q, got %q",
			"demo",
			plan.ManifestName,
		)
	}

	wantModules := []string{
		"logger@v1",
		"http@v1",
		"web@v1",
	}

	if len(plan.Steps) != len(wantModules) {
		t.Fatalf(
			"expected %d steps, got %d",
			len(wantModules),
			len(plan.Steps),
		)
	}

	for i, want := range wantModules {
		if plan.Steps[i].Module != want {
			t.Fatalf(
				"step %d: expected module %q, got %q",
				i,
				want,
				plan.Steps[i].Module,
			)
		}
	}
}

func TestBuildPlanPreservesManifestIdentity(t *testing.T) {
	m := Manifest{
		Version: "v2",
		Name:    "sample-app",
		Modules: []Module{
			{
				Name:    "api",
				Version: "v1",
			},
		},
	}

	plan, err := BuildPlanForManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	if plan.ManifestVersion != m.Version {
		t.Fatalf(
			"expected manifest version %q, got %q",
			m.Version,
			plan.ManifestVersion,
		)
	}

	if plan.ManifestName != m.Name {
		t.Fatalf(
			"expected manifest name %q, got %q",
			m.Name,
			plan.ManifestName,
		)
	}
}

func TestBuildPlanIgnoresApplicationEntrypoint(t *testing.T) {
	withoutEntrypoint := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "app",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "library", Version: "v1"},
				},
			},
			{
				Name:    "library",
				Version: "v1",
			},
		},
	}
	withEntrypoint := withoutEntrypoint
	withEntrypoint.Entrypoint = &ApplicationEntrypoint{
		Module:  "app",
		Version: "v1",
	}

	withoutPlan, err := BuildPlanForManifest(withoutEntrypoint)
	if err != nil {
		t.Fatal(err)
	}
	withPlan, err := BuildPlanForManifest(withEntrypoint)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(withoutPlan, withPlan) {
		t.Fatalf(
			"entrypoint changed build plan:\nwithout: %#v\nwith:    %#v",
			withoutPlan,
			withPlan,
		)
	}
}

func TestBuildPlanUsesDependencyFirstOrder(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	plan, err := BuildPlanForManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"logger@v1",
		"http@v1",
		"web@v1",
	}

	if len(plan.Steps) != len(want) {
		t.Fatalf(
			"expected %d steps, got %d",
			len(want),
			len(plan.Steps),
		)
	}

	for i, step := range plan.Steps {
		if step.Module != want[i] {
			t.Fatalf(
				"step %d: expected %q, got %q",
				i,
				want[i],
				step.Module,
			)
		}
	}
}

func TestBuildPlanPreservesDependencyOrder(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	plan, err := BuildPlanForManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	var webStep BuildStep

	for _, step := range plan.Steps {
		if step.Module == "web@v1" {
			webStep = step
			break
		}
	}

	want := []string{
		"http@v1",
		"logger@v1",
	}

	if !reflect.DeepEqual(webStep.Dependencies, want) {
		t.Fatalf(
			"expected dependencies %#v, got %#v",
			want,
			webStep.Dependencies,
		)
	}
}

func TestBuildPlanSharedDependencyAppearsOnce(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "worker",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	plan, err := BuildPlanForManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"http@v1",
		"logger@v1",
		"web@v1",
		"worker@v1",
	}

	if len(plan.Steps) != len(want) {
		t.Fatalf(
			"expected %d steps, got %d",
			len(want),
			len(plan.Steps),
		)
	}

	for i, step := range plan.Steps {
		if step.Module != want[i] {
			t.Fatalf(
				"step %d: expected %q, got %q",
				i,
				want[i],
				step.Module,
			)
		}
	}

	counts := make(map[string]int, len(plan.Steps))

	for _, step := range plan.Steps {
		counts[step.Module]++
	}

	for module, count := range counts {
		if count != 1 {
			t.Fatalf(
				"module %q appears %d times in build plan",
				module,
				count,
			)
		}
	}
}

func TestBuildPlanRejectsCycle(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "cycle",
		Modules: []Module{
			{
				Name:    "a",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "b", Version: "v1"},
				},
			},
			{
				Name:    "b",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "a", Version: "v1"},
				},
			},
		},
	}

	_, err := BuildPlanForManifest(m)
	if err == nil {
		t.Fatal("expected cycle error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf(
			"expected *errors.Error, got %T",
			err,
		)
	}

	if forgeErr.Code != forgeerrors.CodeInvalidManifest {
		t.Fatalf(
			"expected code %s, got %s",
			forgeerrors.CodeInvalidManifest,
			forgeErr.Code,
		)
	}
}

func TestBuildPlanRejectsMissingDependency(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
				},
			},
		},
	}

	_, err := BuildPlanForManifest(m)
	if err == nil {
		t.Fatal("expected missing dependency error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf(
			"expected *errors.Error, got %T",
			err,
		)
	}

	if forgeErr.Code != forgeerrors.CodeNotFound {
		t.Fatalf(
			"expected code %s, got %s",
			forgeerrors.CodeNotFound,
			forgeErr.Code,
		)
	}
}

func TestBuildPlanAllowsEmptyManifestModules(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "empty",
		Modules: nil,
	}

	plan, err := BuildPlanForManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	if plan.ManifestVersion != m.Version {
		t.Fatalf(
			"expected manifest version %q, got %q",
			m.Version,
			plan.ManifestVersion,
		)
	}

	if plan.ManifestName != m.Name {
		t.Fatalf(
			"expected manifest name %q, got %q",
			m.Name,
			plan.ManifestName,
		)
	}

	if len(plan.Steps) != 0 {
		t.Fatalf(
			"expected 0 build steps, got %d",
			len(plan.Steps),
		)
	}
}

func TestBuildPlanDoesNotMutateInput(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
			},
		},
	}

	original := m

	plan, err := BuildPlanForManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	if plan.ManifestName == "" {
		t.Fatal("expected populated build plan")
	}

	if !reflect.DeepEqual(m, original) {
		t.Fatal("BuildPlanForManifest mutated input manifest")
	}
}

func TestBuildPlanReturnsIndependentDependencySnapshots(t *testing.T) {
	m := Manifest{
		Version: "v1",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
				Dependencies: []Dependency{
					{Name: "http", Version: "v1"},
					{Name: "logger", Version: "v1"},
				},
			},
			{
				Name:    "http",
				Version: "v1",
			},
			{
				Name:    "logger",
				Version: "v1",
			},
		},
	}

	plan, err := BuildPlanForManifest(m)
	if err != nil {
		t.Fatal(err)
	}

	var webIndex int

	for i, step := range plan.Steps {
		if step.Module == "web@v1" {
			webIndex = i
			break
		}
	}

	if len(plan.Steps[webIndex].Dependencies) != 2 {
		t.Fatalf(
			"expected 2 dependencies, got %d",
			len(plan.Steps[webIndex].Dependencies),
		)
	}

	plan.Steps[webIndex].Dependencies[0] = "changed@v9"

	if !reflect.DeepEqual(
		m.Modules[0].Dependencies,
		[]Dependency{
			{Name: "http", Version: "v1"},
			{Name: "logger", Version: "v1"},
		},
	) {
		t.Fatal("build plan dependency slice aliased manifest dependency data")
	}
}

func TestBuildPlanValidatesManifestBeforePlanning(t *testing.T) {
	m := Manifest{
		Version: "",
		Name:    "demo",
		Modules: []Module{
			{
				Name:    "web",
				Version: "v1",
			},
		},
	}

	_, err := BuildPlanForManifest(m)
	if err == nil {
		t.Fatal("expected validation error")
	}

	var forgeErr *forgeerrors.Error
	if !errors.As(err, &forgeErr) {
		t.Fatalf(
			"expected *errors.Error, got %T",
			err,
		)
	}

	if forgeErr.Code != forgeerrors.CodeInvalidManifest {
		t.Fatalf(
			"expected code %s, got %s",
			forgeerrors.CodeInvalidManifest,
			forgeErr.Code,
		)
	}
}
