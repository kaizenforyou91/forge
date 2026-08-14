package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ArtifactBundleDocument is the stable JSON representation of an artifact
// bundle.
//
// The explicit tags make the serialized schema independent from Go field
// naming conventions.
type ArtifactBundleDocument struct {
	ManifestName    string             `json:"manifest_name"`
	ManifestVersion string             `json:"manifest_version"`
	Artifacts       []ArtifactDocument `json:"artifacts"`
}

// ArtifactDocument is the stable serialized representation of one artifact.
type ArtifactDocument struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

// MarshalArtifactBundle serializes an artifact bundle into deterministic JSON.
//
// The artifact order stored in the bundle is preserved exactly.
func MarshalArtifactBundle(bundle ArtifactBundle) ([]byte, error) {
	if err := bundle.Validate(); err != nil {
		return nil, err
	}

	document := ArtifactBundleDocument{
		ManifestName:    bundle.ManifestName,
		ManifestVersion: bundle.ManifestVersion,
		Artifacts:       make([]ArtifactDocument, len(bundle.Artifacts)),
	}

	for i, artifact := range bundle.Artifacts {
		document.Artifacts[i] = ArtifactDocument{
			Module:  artifact.Module,
			Version: artifact.Version,
		}
	}

	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: marshal artifact bundle: %v",
			ErrInvalidArtifactBundle,
			err,
		)
	}

	return data, nil
}

// UnmarshalArtifactBundle deserializes and validates an artifact bundle.
//
// Unknown JSON fields are intentionally ignored so the format can evolve
// without breaking older readers.
func UnmarshalArtifactBundle(data []byte) (ArtifactBundle, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return ArtifactBundle{}, fmt.Errorf(
			"%w: empty JSON document",
			ErrInvalidArtifactBundle,
		)
	}

	var document ArtifactBundleDocument

	if err := json.Unmarshal(data, &document); err != nil {
		return ArtifactBundle{}, fmt.Errorf(
			"%w: invalid JSON: %v",
			ErrInvalidArtifactBundle,
			err,
		)
	}

	bundle := ArtifactBundle{
		ManifestName:    document.ManifestName,
		ManifestVersion: document.ManifestVersion,
		Artifacts:       make([]Artifact, len(document.Artifacts)),
	}

	for i, artifact := range document.Artifacts {
		bundle.Artifacts[i] = Artifact{
			Module:  artifact.Module,
			Version: artifact.Version,
		}
	}

	if err := bundle.Validate(); err != nil {
		return ArtifactBundle{}, err
	}

	return bundle, nil
}
