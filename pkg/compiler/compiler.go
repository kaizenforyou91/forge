package compiler

import "github.com/kaizenforyou91/forge/pkg/manifest"

// Compiler transforms a manifest build plan into deterministic artifacts.
//
// FW-040 defines the compiler contract only.
// Actual compilation and artifact persistence are implemented by later
// compiler milestones.
type Compiler interface {
	Compile(manifest.BuildPlan) ([]Artifact, error)
}
