package runtime

import "github.com/kaizenforyou91/forge/pkg/compiler"

// VerifiedRunnablePackage contains detached, strictly verified runnable
// package evidence. Its executable bytes are never exposed by reference.
type VerifiedRunnablePackage struct {
	packageFormatVersion int
	bundleSchemaVersion  int
	manifestName         string
	manifestVersion      string
	entrypoint           compiler.RuntimeEntrypoint
	importPath           string
	targetOS             string
	targetArch           string
	signerKeyID          string
	executable           []byte
}

func (p VerifiedRunnablePackage) PackageFormatVersion() int {
	return p.packageFormatVersion
}

func (p VerifiedRunnablePackage) BundleSchemaVersion() int {
	return p.bundleSchemaVersion
}

func (p VerifiedRunnablePackage) ManifestName() string {
	return p.manifestName
}

func (p VerifiedRunnablePackage) ManifestVersion() string {
	return p.manifestVersion
}

func (p VerifiedRunnablePackage) Entrypoint() compiler.RuntimeEntrypoint {
	return p.entrypoint
}

func (p VerifiedRunnablePackage) ImportPath() string {
	return p.importPath
}

func (p VerifiedRunnablePackage) TargetOS() string {
	return p.targetOS
}

func (p VerifiedRunnablePackage) TargetArch() string {
	return p.targetArch
}

func (p VerifiedRunnablePackage) SignerKeyID() string {
	return p.signerKeyID
}

// ExecutableBytes returns a fresh copy of the verified executable payload.
func (p VerifiedRunnablePackage) ExecutableBytes() []byte {
	return append([]byte(nil), p.executable...)
}
