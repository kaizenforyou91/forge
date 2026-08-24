package compiler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	if err := validateSupportedPackageMetadata(metadata); err != nil {
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
