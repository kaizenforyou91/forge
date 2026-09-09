package tool

import "strings"

// Authority is immutable caller-supplied execution configuration. Provider
// definitions and the C1 Executor derive from one owned binding snapshot.
// It is not model authorization, persistent authorization, or replay state.
// Stage A admission remains separate from execution; C1 re-authorizes calls.
// Concurrent accessor use is safe; handler thread-safety remains caller-owned.
// The zero value and nil receiver expose no definitions or executor.
type Authority struct {
	definitions []Definition
	executor    *Executor
}

// NewAuthority captures binding definitions and handler function values once.
// Callers must not mutate bindings or parameter slices during construction.
// Mutations after return cannot change the captured configuration. Nil/empty
// bindings create a valid deny-all authority, consistent with NewExecutor.
func NewAuthority(bindings []Binding) (*Authority, error) {
	// Reuse Stage A bounds solely to bound snapshot allocations. All contract
	// validation, including handlers, names, types and duplicates, stays in C1.
	if len(bindings) > maxTools {
		return nil, ErrInvalidExecutor
	}
	snapshot := make([]Binding, len(bindings))
	for i, binding := range bindings {
		if len(binding.Definition.Name) > maxNameBytes || len(binding.Definition.Parameters) > maxParameters {
			return nil, ErrInvalidExecutor
		}
		for _, parameter := range binding.Definition.Parameters {
			if len(parameter.Name) > maxNameBytes {
				return nil, ErrInvalidExecutor
			}
		}
		snapshot[i] = Binding{Definition: authorityDefinitionCopy(binding.Definition), Handler: binding.Handler}
	}
	executor, err := NewExecutor(snapshot)
	if err != nil {
		return nil, err
	}
	definitions := make([]Definition, len(snapshot))
	for i, binding := range snapshot {
		definitions[i] = binding.Definition
	}
	return &Authority{definitions: definitions, executor: executor}, nil
}

// Definitions returns fresh deep copies, never mutable authority internals.
func (a *Authority) Definitions() []Definition {
	if a == nil || a.executor == nil {
		return nil
	}
	definitions := make([]Definition, len(a.definitions))
	for i, definition := range a.definitions {
		definitions[i] = authorityDefinitionCopy(definition)
	}
	return definitions
}

// Executor returns the immutable C1 executor from the same binding snapshot.
// It does not execute anything or provide Stage A admission by itself.
func (a *Authority) Executor() *Executor {
	if a == nil {
		return nil
	}
	return a.executor
}

func authorityDefinitionCopy(definition Definition) Definition {
	parameters := make([]Parameter, len(definition.Parameters))
	for i, parameter := range definition.Parameters {
		parameters[i] = Parameter{Name: strings.Clone(parameter.Name), Type: parameter.Type, Required: parameter.Required}
	}
	return Definition{Name: strings.Clone(definition.Name), Parameters: parameters}
}

func (a Authority) String() string   { return "tool authority (configuration redacted)" }
func (a Authority) GoString() string { return a.String() }
