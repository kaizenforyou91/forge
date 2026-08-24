package compiler

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestPackageMetadataV1(t *testing.T) {
	want := packageMetadataDocument{packageFormatVersionV1, artifactBundleSchemaVersionV1}
	if got := packageMetadataV1(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestPackageMetadataV2(t *testing.T) {
	want := packageMetadataDocument{packageFormatVersionV2, artifactBundleSchemaVersionV2}
	if got := packageMetadataV2(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestCurrentPackageMetadataRemainsV1(t *testing.T) {
	if got, want := currentPackageMetadata(), packageMetadataV1(); got != want {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestMarshalPackageMetadataV1CanonicalBytes(t *testing.T) {
	assertPackageMetadataBytes(t, packageMetadataV1(), []byte(`{"package_format_version":1,"bundle_schema_version":1}`))
}

func TestMarshalPackageMetadataV2CanonicalBytes(t *testing.T) {
	assertPackageMetadataBytes(t, packageMetadataV2(), []byte(`{"package_format_version":2,"bundle_schema_version":2}`))
}

func TestMarshalPackageMetadataIsDeterministicForSupportedPairs(t *testing.T) {
	for _, metadata := range []packageMetadataDocument{packageMetadataV1(), packageMetadataV2()} {
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
}

func TestPackageMetadataRejectsMismatchedVersionPairs(t *testing.T) {
	for _, metadata := range []packageMetadataDocument{
		{packageFormatVersionV1, artifactBundleSchemaVersionV2},
		{packageFormatVersionV2, artifactBundleSchemaVersionV1},
	} {
		t.Run(packageMetadataPairTestName(metadata), func(t *testing.T) {
			_, marshalErr := marshalPackageMetadata(metadata)
			if !errors.Is(marshalErr, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected marshal ErrUnsupportedPackageFormat, got %v", marshalErr)
			}

			_, unmarshalErr := unmarshalPackageMetadata([]byte(packageMetadataJSON(metadata.PackageFormatVersion, metadata.BundleSchemaVersion)))
			if !errors.Is(unmarshalErr, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected unmarshal ErrUnsupportedPackageFormat, got %v", unmarshalErr)
			}
		})
	}
}

func TestPackageMetadataRejectsUnsupportedPackageVersions(t *testing.T) {
	for _, version := range []int{0, -1, packageFormatVersionV2 + 1} {
		t.Run(versionTestName(version), func(t *testing.T) {
			metadata := packageMetadataV1()
			metadata.PackageFormatVersion = version

			_, err := marshalPackageMetadata(metadata)
			if !errors.Is(err, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
			}
		})
	}
}

func TestPackageMetadataRejectsUnsupportedBundleVersions(t *testing.T) {
	for _, version := range []int{0, -1, artifactBundleSchemaVersionV2 + 1} {
		t.Run(versionTestName(version), func(t *testing.T) {
			metadata := packageMetadataV1()
			metadata.BundleSchemaVersion = version

			_, err := marshalPackageMetadata(metadata)
			if !errors.Is(err, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
			}
		})
	}
}

func TestUnmarshalPackageMetadataAcceptsSupportedVersionPairs(t *testing.T) {
	for _, want := range []packageMetadataDocument{packageMetadataV1(), packageMetadataV2()} {
		t.Run(packageMetadataPairTestName(want), func(t *testing.T) {
			got, err := unmarshalPackageMetadata([]byte(packageMetadataJSON(want.PackageFormatVersion, want.BundleSchemaVersion)))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("expected %#v, got %#v", want, got)
			}
		})
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
	for _, version := range []int{-1, 0, packageFormatVersionV2 + 1} {
		t.Run(versionTestName(version), func(t *testing.T) {
			data := []byte(packageMetadataJSON(version, artifactBundleSchemaVersionV1))

			_, err := unmarshalPackageMetadata(data)
			if !errors.Is(err, ErrUnsupportedPackageFormat) {
				t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
			}
		})
	}
}

func TestUnmarshalPackageMetadataRejectsUnsupportedBundleSchema(t *testing.T) {
	for _, version := range []int{-1, 0, artifactBundleSchemaVersionV2 + 1} {
		t.Run(versionTestName(version), func(t *testing.T) {
			data := []byte(packageMetadataJSON(packageFormatVersionV1, version))

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

func packageMetadataPairTestName(metadata packageMetadataDocument) string {
	return "package_" + strconv.Itoa(metadata.PackageFormatVersion) +
		"_bundle_" + strconv.Itoa(metadata.BundleSchemaVersion)
}

func assertPackageMetadataBytes(t *testing.T, metadata packageMetadataDocument, want []byte) {
	t.Helper()

	got, err := marshalPackageMetadata(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("expected %s, got %s", want, got)
	}
}
