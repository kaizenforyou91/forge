package validation

import (
	"errors"

	"github.com/kaizenforyou91/forge/pkg/manifest"
)

var ErrNilValidator = errors.New("validator is nil")

// Engine executes manifest validation rules in deterministic order.
//
// Structural manifest validation is always executed first through
// manifest.Manifest.Validate(). Registered validators are then executed
// in registration order.
type Engine struct {
	validators []Validator
}

// NewEngine creates a validation engine.
func NewEngine(validators ...Validator) (*Engine, error) {
	engine := &Engine{
		validators: make([]Validator, 0, len(validators)),
	}

	for _, validator := range validators {
		if err := engine.Add(validator); err != nil {
			return nil, err
		}
	}

	return engine, nil
}

// Add registers a validator.
//
// Validators execute in registration order.
func (e *Engine) Add(validator Validator) error {
	if validator == nil {
		return ErrNilValidator
	}

	e.validators = append(e.validators, validator)

	return nil
}

// Count returns the number of registered validators.
func (e *Engine) Count() int {
	return len(e.validators)
}

// Validate validates a manifest.
//
// Structural validation runs first. If it succeeds, registered validators
// execute in registration order. Validation stops at the first error.
func (e *Engine) Validate(m manifest.Manifest) error {
	if err := m.Validate(); err != nil {
		return err
	}

	for _, validator := range e.validators {
		if err := validator.Validate(cloneManifest(m)); err != nil {
			return err
		}
	}

	return nil
}

func cloneManifest(original manifest.Manifest) manifest.Manifest {
	clone := original

	if original.Entrypoint != nil {
		entrypoint := *original.Entrypoint
		clone.Entrypoint = &entrypoint
	}

	if original.Modules == nil {
		return clone
	}

	clone.Modules = make([]manifest.Module, len(original.Modules))
	for i, module := range original.Modules {
		clone.Modules[i] = module
		if module.Dependencies == nil {
			continue
		}
		clone.Modules[i].Dependencies = make(
			[]manifest.Dependency,
			len(module.Dependencies),
		)
		copy(clone.Modules[i].Dependencies, module.Dependencies)
	}

	return clone
}
