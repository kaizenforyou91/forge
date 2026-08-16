package compiler

// PackageVerificationPolicy defines how Forge validates a package.
type PackageVerificationPolicy struct {
	RequireIntegrity bool
	RequireSignature bool
}

// DefaultPackageVerificationPolicy returns the default Forge policy.
//
// Integrity is required.
// Signature is optional.
func DefaultPackageVerificationPolicy() PackageVerificationPolicy {
	return PackageVerificationPolicy{
		RequireIntegrity: true,
		RequireSignature: false,
	}
}

// StrictPackageVerificationPolicy returns a strict security policy.
//
// Both integrity and signature are required.
func StrictPackageVerificationPolicy() PackageVerificationPolicy {
	return PackageVerificationPolicy{
		RequireIntegrity: true,
		RequireSignature: true,
	}
}

// Validate validates the policy configuration.
func (p PackageVerificationPolicy) Validate() error {
	// Current policy fields are boolean and therefore structurally valid.
	// This method exists as an explicit contract so future policy fields
	// can add validation without changing callers.
	return nil
}
