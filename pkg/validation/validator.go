package validation

import "github.com/kaizenforyou91/forge/pkg/manifest"

// Validator represents a manifest validation rule.
//
// Validators are intentionally focused on semantic or higher-level
// validation. Structural manifest validation remains owned by pkg/manifest.
type Validator interface {
	Validate(manifest.Manifest) error
}

// ValidatorFunc adapts a function into a Validator.
type ValidatorFunc func(manifest.Manifest) error

// Validate executes the wrapped validation function.
func (f ValidatorFunc) Validate(m manifest.Manifest) error {
	return f(m)
}
