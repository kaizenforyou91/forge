package compiler

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

// Engine executes a manifest build plan through an Executor
// and optionally persists generated artifacts through an ArtifactWriter.
type Engine struct {
	executor       Executor
	writer         ArtifactWriter
	sourceResolver PackageSourceResolver
}

// NewEngine creates a compiler execution engine.
//
// This constructor preserves the FW-041 execution-only behavior.
func NewEngine(executor Executor) (*Engine, error) {
	return NewEngineWithWriter(executor, nil)
}

// NewEngineWithWriter creates a compiler execution engine with
// optional artifact persistence.
func NewEngineWithWriter(
	executor Executor,
	writer ArtifactWriter,
) (*Engine, error) {
	if executor == nil {
		return nil, ErrNilExecutor
	}

	return &Engine{
		executor: executor,
		writer:   writer,
	}, nil
}

// Compile executes every build step in deterministic plan order.
//
// If an ArtifactWriter is configured, each artifact is persisted
// immediately after its corresponding execution succeeds.
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

		importPath := ""

		if e.sourceResolver != nil {
			source, err := e.sourceResolver.Resolve(module, version)
			if err != nil {
				return nil, err
			}

			importPath = source.ImportPath
		}

		result, err := e.executor.Execute(ExecutionRequest{
			Module:       step.Module,
			Dependencies: append([]string(nil), step.Dependencies...),
			ImportPath:   importPath,
		})
		if err != nil {
			return nil, err
		}

		artifact := Artifact{
			Module:     result.Module,
			Version:    result.Version,
			ImportPath: result.ImportPath,
		}

		if artifact.Module == "" {
			artifact.Module = module
		}

		if artifact.Version == "" {
			artifact.Version = version
		}

		if e.writer != nil {
			payload := artifactPayload(artifact)

			if err := e.writer.Write(artifact, payload); err != nil {
				return nil, err
			}
		}

		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

func artifactPayload(artifact Artifact) []byte {
	return []byte(artifact.Module + "@" + artifact.Version)
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

func NewEngineWithSourceResolver(
	executor Executor,
	resolver PackageSourceResolver,
) (*Engine, error) {
	if executor == nil {
		return nil, ErrNilExecutor
	}

	return &Engine{
		executor:       executor,
		sourceResolver: resolver,
	}, nil
}
