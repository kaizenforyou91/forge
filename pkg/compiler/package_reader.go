package compiler

// PackageReader reads a packaged artifact bundle and its payloads.
type PackageReader interface {
	Read(path string) (ArtifactBundle, map[string][]byte, error)
}
