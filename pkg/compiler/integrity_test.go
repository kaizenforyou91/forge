package compiler

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func integrityTestBundle() ArtifactBundle {
	return ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:     "logger",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/logger",
				Version:    "v1",
			},
			{
				Module:     "http",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
				Version:    "v1",
			},
			{
				Module:     "web",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/router",
				Version:    "v1",
			},
		},
	}
}

func integrityTestBundleJSON(t *testing.T) []byte {
	t.Helper()

	data, err := MarshalArtifactBundle(integrityTestBundle())
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func integrityTestPayloads() map[string][]byte {
	return map[string][]byte{
		"logger@v1": []byte("logger-artifact"),
		"http@v1":   []byte("http-artifact"),
		"web@v1":    []byte("web-artifact"),
	}
}

func integrityTestPackageMetadataJSON(t *testing.T) []byte {
	t.Helper()

	data, err := marshalPackageMetadata(currentPackageMetadata())
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func buildTestPackageIntegrity(
	bundle ArtifactBundle,
	bundleJSON []byte,
	payloads map[string][]byte,
) (PackageIntegrity, error) {
	metadataJSON, err := marshalPackageMetadata(currentPackageMetadata())
	if err != nil {
		return PackageIntegrity{}, err
	}

	return BuildPackageIntegrity(
		metadataJSON,
		bundle,
		bundleJSON,
		payloads,
	)
}

func verifyTestPackageIntegrity(
	bundle ArtifactBundle,
	bundleJSON []byte,
	payloads map[string][]byte,
	integrity PackageIntegrity,
) error {
	metadataJSON, err := marshalPackageMetadata(currentPackageMetadata())
	if err != nil {
		return err
	}

	return VerifyPackageIntegrity(
		metadataJSON,
		bundle,
		bundleJSON,
		payloads,
		integrity,
	)
}

func TestBuildPackageIntegrity(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	if integrity.Version != 2 {
		t.Fatalf(
			"expected integrity version 2, got %d",
			integrity.Version,
		)
	}

	if integrity.PackageMetadataSHA256 != sha256Hex(
		integrityTestPackageMetadataJSON(t),
	) {
		t.Fatal("expected package metadata SHA-256")
	}

	if integrity.Algorithm != "sha256" {
		t.Fatalf(
			"expected sha256 algorithm, got %q",
			integrity.Algorithm,
		)
	}

	if integrity.BundleSHA256 == "" {
		t.Fatal("expected bundle SHA-256")
	}

	if len(integrity.Artifacts) != len(bundle.Artifacts) {
		t.Fatalf(
			"expected %d artifact digests, got %d",
			len(bundle.Artifacts),
			len(integrity.Artifacts),
		)
	}

	for i, artifact := range bundle.Artifacts {
		got := integrity.Artifacts[i]

		if got.Module != artifact.Module {
			t.Fatalf(
				"artifact %d module: expected %q, got %q",
				i,
				artifact.Module,
				got.Module,
			)
		}

		if got.Version != artifact.Version {
			t.Fatalf(
				"artifact %d version: expected %q, got %q",
				i,
				artifact.Version,
				got.Version,
			)
		}

		if !validSHA256(got.SHA256) {
			t.Fatalf(
				"artifact %d has invalid SHA-256 %q",
				i,
				got.SHA256,
			)
		}
	}
}

func TestBuildPackageIntegrityUsesVersionTwo(t *testing.T) {
	integrity, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if integrity.Version != 2 {
		t.Fatalf("expected version 2, got %d", integrity.Version)
	}
}

func TestBuildPackageIntegrityIncludesPackageMetadataDigest(t *testing.T) {
	metadataJSON := []byte(`{"package_format_version":1,"bundle_schema_version":1}`)
	integrity, err := BuildPackageIntegrity(
		metadataJSON,
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := integrity.PackageMetadataSHA256, sha256Hex(metadataJSON); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildPackageIntegrityRejectsNilPackageMetadata(t *testing.T) {
	_, err := BuildPackageIntegrity(
		nil,
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf("expected ErrInvalidPackageIntegrity, got %v", err)
	}
}

func TestPackageIntegrityValidateRejectsMissingPackageMetadataDigest(t *testing.T) {
	integrity, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}
	integrity.PackageMetadataSHA256 = ""

	if err := integrity.Validate(); !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf("expected ErrInvalidPackageIntegrity, got %v", err)
	}
}

func TestPackageIntegrityValidateRejectsInvalidPackageMetadataDigest(t *testing.T) {
	for _, digest := range []string{"short", strings.Repeat("z", 64)} {
		integrity, err := buildTestPackageIntegrity(
			integrityTestBundle(),
			integrityTestBundleJSON(t),
			integrityTestPayloads(),
		)
		if err != nil {
			t.Fatal(err)
		}
		integrity.PackageMetadataSHA256 = digest

		if err := integrity.Validate(); !errors.Is(err, ErrInvalidPackageIntegrity) {
			t.Fatalf("expected ErrInvalidPackageIntegrity, got %v", err)
		}
	}
}

func TestBuildPackageIntegrityIsDeterministic(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	first, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	second, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(first, second) {
		t.Fatalf(
			"expected deterministic integrity data, first=%#v second=%#v",
			first,
			second,
		)
	}
}

func TestBuildPackageIntegrityPreservesArtifactOrder(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	for i, artifact := range bundle.Artifacts {
		if integrity.Artifacts[i].Module != artifact.Module ||
			integrity.Artifacts[i].Version != artifact.Version {
			t.Fatalf(
				"artifact %d order mismatch",
				i,
			)
		}
	}
}

func TestBuildPackageIntegrityRejectsNilBundleJSON(t *testing.T) {
	_, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		nil,
		integrityTestPayloads(),
	)
	if err == nil {
		t.Fatal("expected nil bundle JSON error")
	}

	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf(
			"expected ErrInvalidPackageIntegrity, got %v",
			err,
		)
	}
}

func TestBuildPackageIntegrityRejectsMissingPayload(t *testing.T) {
	payloads := integrityTestPayloads()
	delete(payloads, "http@v1")

	_, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		payloads,
	)
	if err == nil {
		t.Fatal("expected missing payload error")
	}

	if !errors.Is(err, ErrMissingArtifactPayload) {
		t.Fatalf(
			"expected ErrMissingArtifactPayload, got %v",
			err,
		)
	}
}

func TestBuildPackageIntegrityRejectsUnexpectedPayload(t *testing.T) {
	payloads := integrityTestPayloads()
	payloads["unexpected@v1"] = []byte("unexpected")

	_, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		payloads,
	)
	if err == nil {
		t.Fatal("expected unexpected payload error")
	}

	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf(
			"expected ErrInvalidArtifactPackage, got %v",
			err,
		)
	}
}

func TestPackageIntegrityValidate(t *testing.T) {
	integrity, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := integrity.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPackageIntegrityValidateRejectsBadVersion(t *testing.T) {
	integrity := PackageIntegrity{
		Version:      99,
		Algorithm:    "sha256",
		BundleSHA256: strings.Repeat("a", 64),
	}

	err := integrity.Validate()
	if err == nil {
		t.Fatal("expected invalid version error")
	}

	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf(
			"expected ErrInvalidPackageIntegrity, got %v",
			err,
		)
	}
}

func TestPackageIntegrityValidateRejectsBadAlgorithm(t *testing.T) {
	integrity := PackageIntegrity{
		Version:      1,
		Algorithm:    "sha512",
		BundleSHA256: strings.Repeat("a", 64),
	}

	err := integrity.Validate()
	if err == nil {
		t.Fatal("expected invalid algorithm error")
	}

	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf(
			"expected ErrInvalidPackageIntegrity, got %v",
			err,
		)
	}
}

func TestPackageIntegrityValidateRejectsBadBundleDigest(t *testing.T) {
	integrity := PackageIntegrity{
		Version:      1,
		Algorithm:    "sha256",
		BundleSHA256: "invalid",
	}

	err := integrity.Validate()
	if err == nil {
		t.Fatal("expected invalid bundle digest error")
	}

	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf(
			"expected ErrInvalidPackageIntegrity, got %v",
			err,
		)
	}
}

func TestPackageIntegrityValidateRejectsBadArtifactDigest(t *testing.T) {
	integrity := PackageIntegrity{
		Version:      1,
		Algorithm:    "sha256",
		BundleSHA256: strings.Repeat("a", 64),
		Artifacts: []ArtifactDigest{
			{
				Module:  "http",
				Version: "v1",
				SHA256:  "invalid",
			},
		},
	}

	err := integrity.Validate()
	if err == nil {
		t.Fatal("expected invalid artifact digest error")
	}

	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf(
			"expected ErrInvalidPackageIntegrity, got %v",
			err,
		)
	}
}

func TestPackageIntegrityValidateRejectsDuplicateArtifact(t *testing.T) {
	integrity := PackageIntegrity{
		Version:      1,
		Algorithm:    "sha256",
		BundleSHA256: strings.Repeat("a", 64),
		Artifacts: []ArtifactDigest{
			{
				Module:  "http",
				Version: "v1",
				SHA256:  strings.Repeat("a", 64),
			},
			{
				Module:  "http",
				Version: "v1",
				SHA256:  strings.Repeat("b", 64),
			},
		},
	}

	err := integrity.Validate()
	if err == nil {
		t.Fatal("expected duplicate artifact error")
	}

	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf(
			"expected ErrInvalidPackageIntegrity, got %v",
			err,
		)
	}
}

func TestMarshalPackageIntegrity(t *testing.T) {
	integrity, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		t.Fatal("expected non-empty JSON")
	}

	var decoded PackageIntegrity

	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(decoded, integrity) {
		t.Fatalf(
			"decoded integrity differs: expected=%#v got=%#v",
			integrity,
			decoded,
		)
	}
}

func TestMarshalPackageIntegrityIsDeterministic(t *testing.T) {
	integrity, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	first, err := MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}

	second, err := MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf(
			"expected deterministic JSON:\nfirst:  %s\nsecond: %s",
			first,
			second,
		)
	}
}

func TestMarshalPackageIntegrityVersionTwo(t *testing.T) {
	integrity, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}

	prefix := []byte(`{"version":2,"algorithm":"sha256","package_metadata_sha256":"`)
	if !bytes.HasPrefix(data, prefix) {
		t.Fatalf("expected canonical v2 prefix %s, got %s", prefix, data)
	}
}

func TestUnmarshalPackageIntegrity(t *testing.T) {
	original, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := MarshalPackageIntegrity(original)
	if err != nil {
		t.Fatal(err)
	}

	got, err := UnmarshalPackageIntegrity(data)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, original) {
		t.Fatalf(
			"expected %#v, got %#v",
			original,
			got,
		)
	}
}

func TestUnmarshalPackageIntegrityAcceptsVersionTwo(t *testing.T) {
	integrity, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}

	got, err := UnmarshalPackageIntegrity(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 2 {
		t.Fatalf("expected version 2, got %d", got.Version)
	}
}

func TestUnmarshalPackageIntegrityRejectsVersionOne(t *testing.T) {
	integrity, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}
	integrity.Version = 1

	data, err := json.Marshal(integrity)
	if err != nil {
		t.Fatal(err)
	}

	_, err = UnmarshalPackageIntegrity(data)
	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf("expected ErrInvalidPackageIntegrity, got %v", err)
	}
}

func TestUnmarshalPackageIntegrityRejectsInvalidJSON(t *testing.T) {
	_, err := UnmarshalPackageIntegrity(
		[]byte(`{"version":`),
	)
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}

	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf(
			"expected ErrInvalidPackageIntegrity, got %v",
			err,
		)
	}
}

func TestUnmarshalPackageIntegrityRejectsEmptyDocument(t *testing.T) {
	_, err := UnmarshalPackageIntegrity([]byte("   "))
	if err == nil {
		t.Fatal("expected empty document error")
	}

	if !errors.Is(err, ErrInvalidPackageIntegrity) {
		t.Fatalf(
			"expected ErrInvalidPackageIntegrity, got %v",
			err,
		)
	}
}

func TestUnmarshalPackageIntegrityCreatesIndependentSlice(t *testing.T) {
	original, err := buildTestPackageIntegrity(
		integrityTestBundle(),
		integrityTestBundleJSON(t),
		integrityTestPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}

	data, err := MarshalPackageIntegrity(original)
	if err != nil {
		t.Fatal(err)
	}

	got, err := UnmarshalPackageIntegrity(data)
	if err != nil {
		t.Fatal(err)
	}

	got.Artifacts[0].Module = "changed"

	if original.Artifacts[0].Module == "changed" {
		t.Fatal("integrity artifacts unexpectedly aliased")
	}
}

func TestVerifyPackageIntegrity(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifyTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
		integrity,
	); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyPackageIntegrityRejectsBundleTampering(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	tamperedJSON := bytes.Replace(
		bundleJSON,
		[]byte(`"manifest_version":"v1"`),
		[]byte(`"manifest_version":"v2"`),
		1,
	)

	if bytes.Equal(tamperedJSON, bundleJSON) {
		t.Fatal("failed to tamper with bundle JSON")
	}

	err = verifyTestPackageIntegrity(
		bundle,
		tamperedJSON,
		payloads,
		integrity,
	)
	if err == nil {
		t.Fatal("expected bundle integrity mismatch")
	}

	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf(
			"expected ErrIntegrityMismatch, got %v",
			err,
		)
	}
}

func TestVerifyPackageIntegrityDetectsPackageMetadataMismatch(t *testing.T) {
	metadataJSON := integrityTestPackageMetadataJSON(t)
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()
	integrity, err := BuildPackageIntegrity(
		metadataJSON,
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	tamperedMetadata := append(append([]byte(nil), metadataJSON...), ' ')
	err = VerifyPackageIntegrity(
		tamperedMetadata,
		bundle,
		bundleJSON,
		payloads,
		integrity,
	)
	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
	}
}

func TestChangingOnlyPackageMetadataChangesIntegrityDigest(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	first, err := BuildPackageIntegrity(
		[]byte(`{"package_format_version":1,"bundle_schema_version":1}`),
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildPackageIntegrity(
		[]byte("{\"package_format_version\":1,\"bundle_schema_version\":1} "),
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	if first.PackageMetadataSHA256 == second.PackageMetadataSHA256 {
		t.Fatal("expected metadata digests to differ")
	}
	if first.BundleSHA256 != second.BundleSHA256 {
		t.Fatal("expected bundle digest to remain unchanged")
	}
	if !reflect.DeepEqual(first.Artifacts, second.Artifacts) {
		t.Fatal("expected artifact digests to remain unchanged")
	}
}

func TestVerifyPackageIntegrityRejectsImportPathTampering(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON, err := MarshalArtifactBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}

	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	tamperedBundle := bundle
	tamperedBundle.Artifacts = append([]Artifact(nil), bundle.Artifacts...)
	tamperedBundle.Artifacts[0].ImportPath =
		"github.com/kaizenforyou91/forge/pkg/tampered"

	tamperedBundleJSON, err := MarshalArtifactBundle(tamperedBundle)
	if err != nil {
		t.Fatal(err)
	}

	err = verifyTestPackageIntegrity(
		tamperedBundle,
		tamperedBundleJSON,
		payloads,
		integrity,
	)
	if err == nil {
		t.Fatal("expected import path integrity mismatch")
	}

	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf(
			"expected ErrIntegrityMismatch, got %v",
			err,
		)
	}
}

func TestVerifyPackageIntegrityRejectsPayloadTampering(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	tampered := make(map[string][]byte, len(payloads))

	for key, payload := range payloads {
		tampered[key] = append([]byte(nil), payload...)
	}

	tampered["http@v1"][0] = 'X'

	err = verifyTestPackageIntegrity(
		bundle,
		bundleJSON,
		tampered,
		integrity,
	)
	if err == nil {
		t.Fatal("expected payload integrity mismatch")
	}

	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf(
			"expected ErrIntegrityMismatch, got %v",
			err,
		)
	}
}

func TestVerifyPackageIntegrityRejectsArtifactReordering(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	reorderedBundle := bundle
	reorderedBundle.Artifacts = append(
		[]Artifact(nil),
		bundle.Artifacts...,
	)

	reorderedBundle.Artifacts[0], reorderedBundle.Artifacts[1] =
		reorderedBundle.Artifacts[1],
		reorderedBundle.Artifacts[0]

	err = verifyTestPackageIntegrity(
		reorderedBundle,
		bundleJSON,
		payloads,
		integrity,
	)
	if err == nil {
		t.Fatal("expected artifact order mismatch")
	}

	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf(
			"expected ErrIntegrityMismatch, got %v",
			err,
		)
	}
}

func TestVerifyPackageIntegrityRejectsMissingPayload(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	delete(payloads, "http@v1")

	err = verifyTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
		integrity,
	)
	if err == nil {
		t.Fatal("expected missing payload error")
	}

	if !errors.Is(err, ErrMissingArtifactPayload) {
		t.Fatalf(
			"expected ErrMissingArtifactPayload, got %v",
			err,
		)
	}
}

func TestVerifyPackageIntegrityRejectsUnexpectedPayload(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	payloads["unexpected@v1"] = []byte("unexpected")

	err = verifyTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
		integrity,
	)
	if err == nil {
		t.Fatal("expected unexpected payload error")
	}

	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf(
			"expected ErrIntegrityMismatch, got %v",
			err,
		)
	}
}

func TestVerifyPackageIntegrityRejectsIntegrityArtifactMismatch(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	integrity.Artifacts[0].Module = "changed"

	err = verifyTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
		integrity,
	)
	if err == nil {
		t.Fatal("expected integrity metadata mismatch")
	}

	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf(
			"expected ErrIntegrityMismatch, got %v",
			err,
		)
	}
}

func TestVerifyPackageIntegrityDoesNotMutateInputs(t *testing.T) {
	bundle := integrityTestBundle()
	bundleJSON := integrityTestBundleJSON(t)
	payloads := integrityTestPayloads()

	integrity, err := buildTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
	)
	if err != nil {
		t.Fatal(err)
	}

	originalBundle := bundle
	originalBundleJSON := append([]byte(nil), bundleJSON...)

	originalPayloads := make(map[string][]byte, len(payloads))
	for key, payload := range payloads {
		originalPayloads[key] = append([]byte(nil), payload...)
	}

	originalIntegrity := integrity

	if err := verifyTestPackageIntegrity(
		bundle,
		bundleJSON,
		payloads,
		integrity,
	); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(bundle, originalBundle) {
		t.Fatal("bundle was mutated")
	}

	if !bytes.Equal(bundleJSON, originalBundleJSON) {
		t.Fatal("bundle JSON was mutated")
	}

	if !reflect.DeepEqual(payloads, originalPayloads) {
		t.Fatal("payloads were mutated")
	}

	if !reflect.DeepEqual(integrity, originalIntegrity) {
		t.Fatal("integrity metadata was mutated")
	}
}
