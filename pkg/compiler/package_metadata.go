package compiler

import (
	"encoding/json"
	"fmt"
)

const (
	packageFormatVersionV1 = 1
	packageFormatVersionV2 = 2

	artifactBundleSchemaVersionV1 = 1
	artifactBundleSchemaVersionV2 = 2
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
	// The production writer remains on v1 until runnable package writing is
	// introduced by a coordinated writer/reader migration.
	return packageMetadataV1()
}

func packageMetadataV1() packageMetadataDocument {
	return packageMetadataDocument{
		PackageFormatVersion: packageFormatVersionV1,
		BundleSchemaVersion:  artifactBundleSchemaVersionV1,
	}
}

func packageMetadataV2() packageMetadataDocument {
	return packageMetadataDocument{
		PackageFormatVersion: packageFormatVersionV2,
		BundleSchemaVersion:  artifactBundleSchemaVersionV2,
	}
}

func marshalPackageMetadata(
	metadata packageMetadataDocument,
) ([]byte, error) {
	if err := validateSupportedPackageMetadata(metadata); err != nil {
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
	var wire packageMetadataWire
	if err := decodeStrictJSON(data, &wire); err != nil {
		return packageMetadataDocument{}, fmt.Errorf(
			"%w: decode package metadata: %v",
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

	if err := validateSupportedPackageMetadata(metadata); err != nil {
		return packageMetadataDocument{}, err
	}

	return metadata, nil
}
func validateSupportedPackageMetadata(metadata packageMetadataDocument) error {
	if metadata == packageMetadataV1() || metadata == packageMetadataV2() {
		return nil
	}

	return fmt.Errorf(
		"%w: package format version %d / bundle schema version %d; supported pairs are (1,1) and (2,2)",
		ErrUnsupportedPackageFormat,
		metadata.PackageFormatVersion,
		metadata.BundleSchemaVersion,
	)
}
