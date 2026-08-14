package compiler

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

// Engine executes a manifest build plan through an Executor.
type Engine struct {
	executor Executor
}

// NewEngine creates a compiler execution engine.
func NewEngine(executor Executor) (*Engine, error) {
	if executor == nil {
		return nil, ErrNilExecutor
	}

	return &Engine{
		executor: executor,
	}, nil
}

// Compile executes every build step in deterministic plan order.
func (e *Engine) Compile(plan manifest.BuildPlan) ([]Artifact, error) {
	if e == nil || e.executor == nil {
		return nil, ErrNilExecutor
	}

	artifacts := make([]Artifact, 0, len(plan.Steps))

	for _, step := range plan.Steps {
		module, version, ok := splitModuleIdentity(step.Module)
		if !ok {
			return nil, fmt.Errorf(
				"%w: invalid module identity %q",
				ErrInvalidBuildPlan,
				step.Module,
			)
		}

		result, err := e.executor.Execute(ExecutionRequest{
			Module:       step.Module,
			Dependencies: append([]string(nil), step.Dependencies...),
		})
		if err != nil {
			return nil, err
		}

		artifact := Artifact{
			Module:  result.Module,
			Version: result.Version,
		}

		if artifact.Module == "" {
			artifact.Module = module
		}

		if artifact.Version == "" {
			artifact.Version = version
		}

		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func splitModuleIdentity(key string) (string, string, bool) {
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
