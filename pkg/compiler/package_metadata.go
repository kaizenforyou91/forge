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
