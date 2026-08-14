package compiler

import (
	"github.com/kaizenforyou91/forge/pkg/manifest"
	"github.com/kaizenforyou91/forge/pkg/registry"
)

// CompileManifest resolves a manifest against the package registry,
// creates a deterministic build plan, and compiles it through the engine.
//
// Resolution is performed before executor invocation, ensuring that
// missing packages or invalid dependency graphs fail before compilation.
func (e *Engine) CompileManifest(
	m manifest.Manifest,
	packages *registry.Registry,
) ([]Artifact, error) {
	if e == nil || e.executor == nil {
		return nil, ErrNilExecutor
	}

	resolved, err := manifest.ResolveDependencies(m, packages)
	if err != nil {
		return nil, err
	}

	plan, err := manifest.BuildPlanForManifest(resolved)
	if err != nil {
		return nil, err
	}

	return e.Compile(plan)
}
