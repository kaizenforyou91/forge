package manifest

// BuildPlan describes the deterministic build sequence for a manifest.
type BuildPlan struct {
	ManifestVersion string
	ManifestName    string
	Steps           []BuildStep
}

// BuildStep describes one module build step and its direct dependencies.
type BuildStep struct {
	Module       string
	Dependencies []string
}

// BuildPlanForManifest creates a deterministic build plan.
//
// The plan contains one step for every manifest module.
// Steps are ordered dependency-first using ResolveDependencyOrder.
// Direct dependency order is preserved from the manifest declaration.
func BuildPlanForManifest(m Manifest) (BuildPlan, error) {
	if err := m.Validate(); err != nil {
		return BuildPlan{}, err
	}

	graph, err := BuildDependencyGraph(m)
	if err != nil {
		return BuildPlan{}, err
	}

	order, err := ResolveDependencyOrder(m)
	if err != nil {
		return BuildPlan{}, err
	}

	plan := BuildPlan{
		ManifestVersion: m.Version,
		ManifestName:    m.Name,
		Steps:           make([]BuildStep, 0, len(order)),
	}

	for _, module := range order {
		dependencies := graph.DependenciesByKey(module)

		plan.Steps = append(plan.Steps, BuildStep{
			Module:       module,
			Dependencies: dependencies,
		})
	}

	return plan, nil
}
