package compiler

// ArtifactWriter persists compiled artifact data.
type ArtifactWriter interface {
	Write(Artifact, []byte) error
}
