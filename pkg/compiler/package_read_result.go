package compiler

// PackageReadResult contains validated package read evidence.
//
// Bundle, Payloads, and their byte slices are owned by the returned result and
// do not depend on the package file remaining open. VerifiedSignerKeyID is
// populated only when a configured PackageVerifier successfully verifies a
// present package signature. The result is evidence returned by
// ZIPPackageReader; it is not an unforgeable trust token.
type PackageReadResult struct {
	PackageFormatVersion int
	BundleSchemaVersion  int
	Bundle               ArtifactBundle
	Payloads             map[string][]byte
	VerifiedSignerKeyID  string
}
