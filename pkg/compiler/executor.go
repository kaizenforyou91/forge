package compiler

// ExecutionRequest describes one deterministic module compilation request.
type ExecutionRequest struct {
	Module       string
	Dependencies []string
	ImportPath   string
}

// ExecutionResult describes the result returned by an execution backend.
type ExecutionResult struct {
	Module  string
	Version string
}

// Executor executes one compiler request.
//
// FW-041 defines the execution abstraction only.
// Concrete toolchain execution is introduced by a later milestone.
type Executor interface {
	Execute(ExecutionRequest) (ExecutionResult, error)
}
