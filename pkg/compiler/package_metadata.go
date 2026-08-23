package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const (
	currentPackageFormatVersion        = 1
	currentArtifactBundleSchemaVersion = 1
)

// packageMetadataDocument is the serialized package.json contract.
type packageMetadataDocument struct {
	PackageFormatVersion int `json:"package_format_version"`
	BundleSchemaVersion  int `json:"bundle_schema_version"`
}

type packageMetadataWire struct {
	PackageFormatVersion *int `json:"package_format_version"`
	BundleSchemaVersion  *int `json:"bundle_schema_version"`
}

func currentPackageMetadata() packageMetadataDocument {
	return packageMetadataDocument{
		PackageFormatVersion: currentPackageFormatVersion,
		BundleSchemaVersion:  currentArtifactBundleSchemaVersion,
	}
}

func marshalPackageMetadata(
	metadata packageMetadataDocument,
) ([]byte, error) {
	if err := validatePackageMetadata(metadata); err != nil {
		return nil, err
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: marshal package metadata: %v",
			ErrInvalidPackageMetadata,
			err,
		)
	}

	return data, nil
}

func unmarshalPackageMetadata(
	data []byte,
) (packageMetadataDocument, error) {
	if err := rejectDuplicatePackageMetadataKeys(data); err != nil {
		return packageMetadataDocument{}, fmt.Errorf(
			"%w: inspect package metadata object: %v",
			ErrInvalidPackageMetadata,
			err,
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var wire packageMetadataWire
	if err := decoder.Decode(&wire); err != nil {
		return packageMetadataDocument{}, fmt.Errorf(
			"%w: decode package metadata: %v",
			ErrInvalidPackageMetadata,
			err,
		)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("unexpected trailing JSON value")
		}

		return packageMetadataDocument{}, fmt.Errorf(
			"%w: trailing package metadata content: %v",
			ErrInvalidPackageMetadata,
			err,
		)
	}

	if wire.PackageFormatVersion == nil {
		return packageMetadataDocument{}, fmt.Errorf(
			"%w: package_format_version is required",
			ErrInvalidPackageMetadata,
		)
	}

	if wire.BundleSchemaVersion == nil {
		return packageMetadataDocument{}, fmt.Errorf(
			"%w: bundle_schema_version is required",
			ErrInvalidPackageMetadata,
		)
	}

	metadata := packageMetadataDocument{
		PackageFormatVersion: *wire.PackageFormatVersion,
		BundleSchemaVersion:  *wire.BundleSchemaVersion,
	}

	if err := validatePackageMetadata(metadata); err != nil {
		return packageMetadataDocument{}, err
	}

	return metadata, nil
}

func rejectDuplicatePackageMetadataKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))

	token, err := decoder.Token()
	if err != nil {
		return err
	}

	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '{' {
		return fmt.Errorf("top-level JSON value must be an object")
	}

	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}

		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("object key must be a string")
		}

		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate object key %q", key)
		}
		seen[key] = struct{}{}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
	}

	token, err = decoder.Token()
	if err != nil {
		return err
	}

	delimiter, ok = token.(json.Delim)
	if !ok || delimiter != '}' {
		return fmt.Errorf("top-level JSON object is not closed")
	}

	if token, err = decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON token %v", token)
		}

		return err
	}

	return nil
}

func validatePackageMetadata(metadata packageMetadataDocument) error {
	if metadata.PackageFormatVersion != currentPackageFormatVersion {
		return fmt.Errorf(
			"%w: package format version %d, supported version %d",
			ErrUnsupportedPackageFormat,
			metadata.PackageFormatVersion,
			currentPackageFormatVersion,
		)
	}

	if metadata.BundleSchemaVersion != currentArtifactBundleSchemaVersion {
		return fmt.Errorf(
			"%w: bundle schema version %d, supported version %d",
			ErrUnsupportedPackageFormat,
			metadata.BundleSchemaVersion,
			currentArtifactBundleSchemaVersion,
		)
	}

	return nil
}
