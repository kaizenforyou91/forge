package compiler

// PackageSignatureState describes the successful signature state of an
// inspected package.
type PackageSignatureState string

const (
	// PackageSignatureUnsigned means the package contains no signature.
	PackageSignatureUnsigned PackageSignatureState = "unsigned"
	// PackageSignatureSignedUnverified means the package signature matches its
	// embedded public key, but that key was not verified as trusted.
	PackageSignatureSignedUnverified PackageSignatureState = "signed_unverified"
	// PackageSignatureSignedTrusted means the package signature was verified
	// against an explicitly supplied trusted key.
	PackageSignatureSignedTrusted PackageSignatureState = "signed_trusted"
)

// PackageInspectionResult contains validated, metadata-only package evidence.
//
// A successful result implies that package integrity was verified. A
// SignedUnverified signature proves only consistency with the public key
// embedded in the package; it does not establish publisher identity or trust.
type PackageInspectionResult struct {
	PackageFormatVersion int
	BundleSchemaVersion  int
	Bundle               ArtifactBundle
	SignatureState       PackageSignatureState
	DeclaredKeyID        string
	VerifiedSignerKeyID  string
}
