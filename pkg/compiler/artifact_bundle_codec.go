package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Module     string `json:"module"`
	Version    string `json:"version"`
	ImportPath string `json:"import_path"`
}

type artifactBundleDocumentV2 struct {
	ManifestName    string                    `json:"manifest_name"`
	ManifestVersion string                    `json:"manifest_version"`
	Runtime         runtimeDescriptorDocument `json:"runtime"`
	Artifacts       []ArtifactDocument        `json:"artifacts"`
}

type runtimeDescriptorDocument struct {
	Kind       string                    `json:"kind"`
	Entrypoint runtimeEntrypointDocument `json:"entrypoint"`
	TargetOS   string                    `json:"target_os"`
	TargetArch string                    `json:"target_arch"`
}

type runtimeEntrypointDocument struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

// MarshalArtifactBundle serializes an artifact bundle into deterministic JSON.
//
// The artifact order stored in the bundle is preserved exactly.
func MarshalArtifactBundle(bundle ArtifactBundle) ([]byte, error) {
	if err := bundle.ValidateForSchema(artifactBundleSchemaVersionV1); err != nil {
		return nil, err
	}

	document := ArtifactBundleDocument{
		ManifestName:    bundle.ManifestName,
		ManifestVersion: bundle.ManifestVersion,
		Artifacts:       make([]ArtifactDocument, len(bundle.Artifacts)),
	}

	for i, artifact := range bundle.Artifacts {
		document.Artifacts[i] = ArtifactDocument{
			Module:     artifact.Module,
			Version:    artifact.Version,
			ImportPath: artifact.ImportPath,
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

func marshalArtifactBundleForSchema(bundle ArtifactBundle, schemaVersion int) ([]byte, error) {
	if schemaVersion == artifactBundleSchemaVersionV1 {
		return MarshalArtifactBundle(bundle)
	}
	if schemaVersion != artifactBundleSchemaVersionV2 {
		return nil, fmt.Errorf(
			"%w: unsupported artifact bundle schema version %d",
			ErrInvalidArtifactBundle,
			schemaVersion,
		)
	}
	if err := bundle.ValidateForSchema(schemaVersion); err != nil {
		return nil, err
	}

	document := artifactBundleDocumentV2{
		ManifestName:    bundle.ManifestName,
		ManifestVersion: bundle.ManifestVersion,
		Runtime: runtimeDescriptorDocument{
			Kind: bundle.Runtime.Kind,
			Entrypoint: runtimeEntrypointDocument{
				Module:  bundle.Runtime.Entrypoint.Module,
				Version: bundle.Runtime.Entrypoint.Version,
			},
			TargetOS:   bundle.Runtime.TargetOS,
			TargetArch: bundle.Runtime.TargetArch,
		},
		Artifacts: artifactDocuments(bundle.Artifacts),
	}

	data, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal artifact bundle: %v", ErrInvalidArtifactBundle, err)
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
			Module:     artifact.Module,
			Version:    artifact.Version,
			ImportPath: artifact.ImportPath,
		}
	}

	if err := bundle.Validate(); err != nil {
		return ArtifactBundle{}, err
	}

	return bundle, nil
}

func unmarshalArtifactBundleForSchema(data []byte, schemaVersion int) (ArtifactBundle, error) {
	if schemaVersion == artifactBundleSchemaVersionV1 {
		return UnmarshalArtifactBundle(data)
	}
	if schemaVersion != artifactBundleSchemaVersionV2 {
		return ArtifactBundle{}, fmt.Errorf(
			"%w: unsupported artifact bundle schema version %d",
			ErrInvalidArtifactBundle,
			schemaVersion,
		)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return ArtifactBundle{}, fmt.Errorf("%w: empty JSON document", ErrInvalidArtifactBundle)
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return ArtifactBundle{}, fmt.Errorf("%w: inspect artifact bundle: %v", ErrInvalidArtifactBundle, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document artifactBundleDocumentV2
	if err := decoder.Decode(&document); err != nil {
		return ArtifactBundle{}, fmt.Errorf("%w: invalid JSON: %v", ErrInvalidArtifactBundle, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("unexpected trailing JSON value")
		}
		return ArtifactBundle{}, fmt.Errorf("%w: trailing JSON content: %v", ErrInvalidArtifactBundle, err)
	}

	bundle := ArtifactBundle{
		ManifestName:    document.ManifestName,
		ManifestVersion: document.ManifestVersion,
		Runtime: &RuntimeDescriptor{
			Kind: document.Runtime.Kind,
			Entrypoint: RuntimeEntrypoint{
				Module:  document.Runtime.Entrypoint.Module,
				Version: document.Runtime.Entrypoint.Version,
			},
			TargetOS:   document.Runtime.TargetOS,
			TargetArch: document.Runtime.TargetArch,
		},
		Artifacts: artifactsFromDocuments(document.Artifacts),
	}
	if err := bundle.ValidateForSchema(schemaVersion); err != nil {
		return ArtifactBundle{}, err
	}
	return bundle, nil
}

func artifactDocuments(artifacts []Artifact) []ArtifactDocument {
	documents := make([]ArtifactDocument, len(artifacts))
	for i, artifact := range artifacts {
		documents[i] = ArtifactDocument{Module: artifact.Module, Version: artifact.Version, ImportPath: artifact.ImportPath}
	}
	return documents
}

func artifactsFromDocuments(documents []ArtifactDocument) []Artifact {
	artifacts := make([]Artifact, len(documents))
	for i, artifact := range documents {
		artifacts[i] = Artifact{Module: artifact.Module, Version: artifact.Version, ImportPath: artifact.ImportPath}
	}
	return artifacts
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := inspectJSONValue(decoder); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON token %v", token)
		}
		return err
	}
	return nil
}

func inspectJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key must be a string")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate object key %q", key)
			}
			seen[key] = struct{}{}
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim('}') {
			return fmt.Errorf("object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := inspectJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil {
			return err
		}
		if closing != json.Delim(']') {
			return fmt.Errorf("array is not closed")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}
