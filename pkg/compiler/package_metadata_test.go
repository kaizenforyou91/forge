package compiler

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestCurrentPackageMetadata(t *testing.T) {
	metadata := currentPackageMetadata()

	if metadata.PackageFormatVersion != 1 {
		t.Fatalf("expected package format version 1, got %d", metadata.PackageFormatVersion)
	}

	if metadata.BundleSchemaVersion != 1 {
		t.Fatalf("expected bundle schema version 1, got %d", metadata.BundleSchemaVersion)
	}
}

func TestMarshalPackageMetadata(t *testing.T) {
	data, err := marshalPackageMetadata(currentPackageMetadata())
	if err != nil {
		t.Fatal(err)
	}

	want := []byte(`{"package_format_version":1,"bundle_schema_version":1}`)
	if !bytes.Equal(data, want) {
		t.Fatalf("expected %s, got %s", want, data)
	}
}

func TestMarshalPackageMetadataIsDeterministic(t *testing.T) {
	metadata := currentPackageMetadata()
	first, err := marshalPackageMetadata(metadata)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		got, err := marshalPackageMetadata(metadata)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(got, first) {
			t.Fatalf("serialization %d differs: first %s, got %s", i, first, got)
		}
	}
}

func TestMarshalPackageMetadataRejectsUnsupportedPackageFormat(t *testing.T) {
	for _, version := range []int{0, -1, currentPackageFormatVersion + 1} {
		t.Run(versionTestName(version), func(t *testing.T) {
			metadata := currentPackageMetadata()
			metadata.PackageFormatVersion = version

			_, err := marshalPackageMetadata(metadata)
			if !errors.Is(err, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
			}
		})
	}
}

func TestMarshalPackageMetadataRejectsUnsupportedBundleSchema(t *testing.T) {
	for _, version := range []int{0, -1, currentArtifactBundleSchemaVersion + 1} {
		t.Run(versionTestName(version), func(t *testing.T) {
			metadata := currentPackageMetadata()
			metadata.BundleSchemaVersion = version

			_, err := marshalPackageMetadata(metadata)
			if !errors.Is(err, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
			}
		})
	}
}

func TestUnmarshalPackageMetadata(t *testing.T) {
	data := []byte(`{"package_format_version":1,"bundle_schema_version":1}`)

	got, err := unmarshalPackageMetadata(data)
	if err != nil {
		t.Fatal(err)
	}

	if want := currentPackageMetadata(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestUnmarshalPackageMetadataRejectsMissingPackageFormatVersion(t *testing.T) {
	_, err := unmarshalPackageMetadata([]byte(`{"bundle_schema_version":1}`))
	if !errors.Is(err, ErrInvalidPackageMetadata) {
		t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
	}
}

func TestUnmarshalPackageMetadataRejectsMissingBundleSchemaVersion(t *testing.T) {
	_, err := unmarshalPackageMetadata([]byte(`{"package_format_version":1}`))
	if !errors.Is(err, ErrInvalidPackageMetadata) {
		t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
	}
}

func TestUnmarshalPackageMetadataRejectsMalformedJSON(t *testing.T) {
	tests := map[string]string{
		"truncated object":                  `{"package_format_version":1`,
		"invalid token":                     `not-json`,
		"wrong package format version type": `{"package_format_version":"1","bundle_schema_version":1}`,
		"wrong bundle schema version type":  `{"package_format_version":1,"bundle_schema_version":{}}`,
	}

	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := unmarshalPackageMetadata([]byte(data))
			if !errors.Is(err, ErrInvalidPackageMetadata) {
				t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
			}
		})
	}
}

func TestUnmarshalPackageMetadataRejectsUnknownField(t *testing.T) {
	data := []byte(`{"package_format_version":1,"bundle_schema_version":1,"required_future_behavior":true}`)

	_, err := unmarshalPackageMetadata(data)
	if !errors.Is(err, ErrInvalidPackageMetadata) {
		t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
	}
}

func TestUnmarshalPackageMetadataRejectsDuplicatePackageFormatVersion(
	t *testing.T,
) {
	data := []byte(`{"package_format_version":1,"package_format_version":2,"bundle_schema_version":1}`)

	_, err := unmarshalPackageMetadata(data)
	if !errors.Is(err, ErrInvalidPackageMetadata) {
		t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
	}
}

func TestUnmarshalPackageMetadataRejectsDuplicateBundleSchemaVersion(
	t *testing.T,
) {
	data := []byte(`{"package_format_version":1,"bundle_schema_version":1,"bundle_schema_version":1}`)

	_, err := unmarshalPackageMetadata(data)
	if !errors.Is(err, ErrInvalidPackageMetadata) {
		t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
	}
}

func TestUnmarshalPackageMetadataRejectsDuplicateKnownFieldBeforeSemanticValidation(
	t *testing.T,
) {
	data := []byte(`{"package_format_version":1,"package_format_version":999,"bundle_schema_version":1}`)

	_, err := unmarshalPackageMetadata(data)
	if !errors.Is(err, ErrInvalidPackageMetadata) {
		t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
	}

	if errors.Is(err, ErrUnsupportedPackageFormat) {
		t.Fatalf("expected structural metadata error, got %v", err)
	}
}

func TestUnmarshalPackageMetadataRejectsDuplicateUnknownField(t *testing.T) {
	data := []byte(`{"package_format_version":1,"bundle_schema_version":1,"future":1,"future":2}`)

	_, err := unmarshalPackageMetadata(data)
	if !errors.Is(err, ErrInvalidPackageMetadata) {
		t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
	}
}

func TestUnmarshalPackageMetadataRejectsUnsupportedPackageFormat(t *testing.T) {
	for _, version := range []int{-1, 0, currentPackageFormatVersion + 1} {
		t.Run(versionTestName(version), func(t *testing.T) {
			data := []byte(packageMetadataJSON(version, currentArtifactBundleSchemaVersion))

			_, err := unmarshalPackageMetadata(data)
			if !errors.Is(err, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
			}
		})
	}
}

func TestUnmarshalPackageMetadataRejectsUnsupportedBundleSchema(t *testing.T) {
	for _, version := range []int{-1, 0, currentArtifactBundleSchemaVersion + 1} {
		t.Run(versionTestName(version), func(t *testing.T) {
			data := []byte(packageMetadataJSON(currentPackageFormatVersion, version))

			_, err := unmarshalPackageMetadata(data)
			if !errors.Is(err, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
			}
		})
	}
}

func TestUnmarshalPackageMetadataRejectsTrailingJSONValue(t *testing.T) {
	data := []byte(`{"package_format_version":1,"bundle_schema_version":1}{}`)

	_, err := unmarshalPackageMetadata(data)
	if !errors.Is(err, ErrInvalidPackageMetadata) {
		t.Fatalf("expected ErrInvalidPackageMetadata, got %v", err)
	}
}

func TestUnmarshalPackageMetadataAllowsTrailingWhitespace(t *testing.T) {
	data := []byte("{\"package_format_version\":1,\"bundle_schema_version\":1} \r\n\t")

	got, err := unmarshalPackageMetadata(data)
	if err != nil {
		t.Fatal(err)
	}

	if want := currentPackageMetadata(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestPackageMetadataErrorSentinelsAreDistinct(t *testing.T) {
	errorsToCompare := []error{
		ErrInvalidPackageMetadata,
		ErrUnsupportedPackageFormat,
		ErrLegacyPackageUnsupported,
	}

	for i, candidate := range errorsToCompare {
		for j, target := range errorsToCompare {
			if i == j {
				continue
			}

			if errors.Is(candidate, target) {
				t.Fatalf("expected %v and %v to be distinct", candidate, target)
			}
		}
	}
}

func versionTestName(version int) string {
	if version < 0 {
		return "negative"
	}

	if version == 0 {
		return "zero"
	}

	return "future"
}

func packageMetadataJSON(packageVersion, bundleVersion int) string {
	return `{"package_format_version":` + strconv.Itoa(packageVersion) +
		`,"bundle_schema_version":` + strconv.Itoa(bundleVersion) + `}`
}
