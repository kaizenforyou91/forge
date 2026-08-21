package compiler

// Artifact represents the result identity of compiling one build-plan module.
//
// The foundation contract intentionally avoids execution details such as
// filesystem paths, binaries, caches, or remote storage.
type Artifact struct {
	Module  string
	Version string
	ImportPath string
}
