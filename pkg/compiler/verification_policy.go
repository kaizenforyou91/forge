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
	// A package signature signs integrity.json. Therefore a policy
	// requiring signatures must also require integrity metadata.
	if p.RequireSignature && !p.RequireIntegrity {
		return ErrInvalidVerificationPolicy
	}

	return nil
}
