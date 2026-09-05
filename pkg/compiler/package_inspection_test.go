package compiler

import (
	"archive/zip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestInspectPackageSignatureStates(t *testing.T) {
	dir := t.TempDir()
	unsignedPath := filepath.Join(dir, "unsigned-v1.zip")
	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		unsignedPath,
	); err != nil {
		t.Fatal(err)
	}

	unsigned, err := InspectPackage(unsignedPath)
	if err != nil {
		t.Fatal(err)
	}
	assertInspectionSignatureState(
		t,
		unsigned,
		PackageSignatureUnsigned,
		"",
		"",
	)

	signer, verifier := inspectionTestSignerAndVerifier(t, "inspection-key")
	signedPath := filepath.Join(dir, "signed-v1.zip")
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		signedPath,
	); err != nil {
		t.Fatal(err)
	}

	unverified, err := InspectPackage(signedPath)
	if err != nil {
		t.Fatal(err)
	}
	assertInspectionSignatureState(
		t,
		unverified,
		PackageSignatureSignedUnverified,
		"inspection-key",
		"",
	)

	trusted, err := InspectPackageWithVerifier(signedPath, verifier)
	if err != nil {
		t.Fatal(err)
	}
	assertInspectionSignatureState(
		t,
		trusted,
		PackageSignatureSignedTrusted,
		"inspection-key",
		"inspection-key",
	)
}

func TestInspectPackageSupportsV1AndCrossHostV2(t *testing.T) {
	dir := t.TempDir()
	v1Path := filepath.Join(dir, "v1.zip")
	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		v1Path,
	); err != nil {
		t.Fatal(err)
	}

	v1, err := InspectPackage(v1Path)
	if err != nil {
		t.Fatal(err)
	}
	if v1.PackageFormatVersion != 1 || v1.BundleSchemaVersion != 1 {
		t.Fatalf("unexpected v1 evidence: %#v", v1)
	}
	if v1.Bundle.Runtime != nil {
		t.Fatalf("v1 inspection exposed runtime metadata: %#v", v1.Bundle.Runtime)
	}

	targetOS := "windows"
	if runtime.GOOS == targetOS {
		targetOS = "linux"
	}
	v2Path := filepath.Join(dir, "cross-host-v2.zip")
	v2Bundle := inspectionTestV2Bundle(targetOS, "amd64")
	if err := NewZIPPackager().packageForMetadata(
		v2Bundle,
		map[string][]byte{"app@v1": []byte("cross-host executable")},
		v2Path,
		packageMetadataV2(),
	); err != nil {
		t.Fatal(err)
	}

	v2, err := InspectPackage(v2Path)
	if err != nil {
		t.Fatalf("cross-host inspection failed: %v", err)
	}
	if v2.PackageFormatVersion != 2 || v2.BundleSchemaVersion != 2 ||
		v2.Bundle.Runtime == nil || v2.Bundle.Runtime.TargetOS != targetOS {
		t.Fatalf("unexpected cross-host v2 evidence: %#v", v2)
	}
}

func TestInspectPackageRejectsSignatureMismatchWithoutChangingReadDetailed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mismatched-signature.zip")
	signer, _ := inspectionTestSignerAndVerifier(t, "self-check")
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, path)
	signature, err := UnmarshalPackageSignature(entries[signatureManifestPath])
	if err != nil {
		t.Fatal(err)
	}
	rawSignature, err := base64.StdEncoding.DecodeString(signature.Signature)
	if err != nil {
		t.Fatal(err)
	}
	rawSignature[0] ^= 0xff
	signature.Signature = base64.StdEncoding.EncodeToString(rawSignature)
	entries[signatureManifestPath], err = MarshalPackageSignature(signature)
	if err != nil {
		t.Fatal(err)
	}
	writeZIPEntriesWithMethodForReadLimitTest(t, path, entries, zip.Store)

	if _, err := InspectPackage(path); !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got %v", err)
	}
	result, err := NewZIPPackageReader().ReadDetailed(path)
	if err != nil {
		t.Fatalf("default ReadDetailed signature behavior changed: %v", err)
	}
	if result.VerifiedSignerKeyID != "" {
		t.Fatalf("default ReadDetailed trusted unexpected signer %q", result.VerifiedSignerKeyID)
	}
}

func TestInspectPackageTrustedModeNeverDowngrades(t *testing.T) {
	dir := t.TempDir()
	unsignedPath := filepath.Join(dir, "unsigned.zip")
	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		unsignedPath,
	); err != nil {
		t.Fatal(err)
	}
	_, verifier := inspectionTestSignerAndVerifier(t, "trusted")
	if _, err := InspectPackageWithVerifier(
		unsignedPath,
		verifier,
	); !errors.Is(err, ErrMissingPackageSignature) {
		t.Fatalf("expected ErrMissingPackageSignature, got %v", err)
	}

	attacker, _ := inspectionTestSignerAndVerifier(t, "attacker")
	signedPath := filepath.Join(dir, "attacker.zip")
	if err := NewZIPPackagerWithSigner(attacker).Package(
		testPackageBundle(),
		testPackagePayloads(),
		signedPath,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := InspectPackageWithVerifier(
		signedPath,
		verifier,
	); !errors.Is(err, ErrUntrustedPackageKey) {
		t.Fatalf("expected ErrUntrustedPackageKey, got %v", err)
	}
	if _, err := InspectPackageWithVerifier(signedPath, nil); !errors.Is(
		err,
		ErrPackageVerifierRequired,
	) {
		t.Fatalf("expected ErrPackageVerifierRequired, got %v", err)
	}
}

func TestInspectPackageRejectsIntegrityTamperAndUnsupportedInputs(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	tamperedEntries := readZIPEntriesForTest(t, validPath)
	tamperedEntries["artifacts/http/v1/artifact"] = []byte("tampered")
	tamperedPath := filepath.Join(dir, "tampered.zip")
	writeZIPEntriesWithMethodForReadLimitTest(
		t,
		tamperedPath,
		tamperedEntries,
		zip.Store,
	)
	if _, err := InspectPackage(tamperedPath); !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
	}

	unsupportedEntries := readZIPEntriesForTest(t, validPath)
	unsupportedEntries[packageMetadataPath] = []byte(
		`{"package_format_version":2,"bundle_schema_version":1}`,
	)
	unsupportedPath := filepath.Join(dir, "unsupported.zip")
	writeZIPEntriesWithMethodForReadLimitTest(
		t,
		unsupportedPath,
		unsupportedEntries,
		zip.Store,
	)
	if _, err := InspectPackage(unsupportedPath); !errors.Is(
		err,
		ErrUnsupportedPackageFormat,
	) {
		t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
	}

	legacyPath := filepath.Join(dir, "legacy.zip")
	writeZIPEntriesWithMethodForReadLimitTest(
		t,
		legacyPath,
		map[string][]byte{bundleManifestPath: []byte(`{}`)},
		zip.Store,
	)
	if _, err := InspectPackage(legacyPath); !errors.Is(
		err,
		ErrLegacyPackageUnsupported,
	) {
		t.Fatalf("expected ErrLegacyPackageUnsupported, got %v", err)
	}
}

func TestInspectPackageEnforcesAllReadLimits(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(validPath)
	if err != nil {
		t.Fatal(err)
	}
	entries := readZIPEntriesForTest(t, validPath)

	tests := map[string]func(*PackageReadLimits){
		"archive": func(limits *PackageReadLimits) {
			limits.MaxArchiveBytes = info.Size() - 1
		},
		"entries": func(limits *PackageReadLimits) {
			limits.MaxEntries = len(entries) - 1
		},
		"document": func(limits *PackageReadLimits) {
			limits.MaxDocumentBytes = 1
		},
		"artifact": func(limits *PackageReadLimits) {
			limits.MaxArtifactBytes = 1
		},
		"total uncompressed": func(limits *PackageReadLimits) {
			limits.MaxTotalUncompressedBytes = 1
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			limits := testPackageReadLimits()
			mutate(&limits)
			_, err := inspectPackageWithVerifierAndLimits(
				validPath,
				nil,
				false,
				limits,
			)
			if !errors.Is(err, ErrPackageReadLimitExceeded) {
				t.Fatalf("expected ErrPackageReadLimitExceeded, got %v", err)
			}
		})
	}

	deflatedPath := filepath.Join(dir, "deflated.zip")
	writeZIPEntriesForTest(t, deflatedPath, entries)
	_, err = inspectPackageWithVerifierAndLimits(
		deflatedPath,
		nil,
		false,
		testPackageReadLimits(),
	)
	if !errors.Is(err, ErrInvalidArtifactPackage) {
		t.Fatalf("expected Store-only rejection, got %v", err)
	}
}

func TestInspectPackageUsesSharedSingleOpenPipeline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package.zip")
	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	); err != nil {
		t.Fatal(err)
	}
	reader := newBoundedPackageReaderForTest(t, testPackageReadLimits())
	opens := 0
	reader.open = func(path string) (*os.File, error) {
		opens++
		return os.Open(path)
	}

	if _, err := inspectPackageWithReader(path, reader, nil, false); err != nil {
		t.Fatal(err)
	}
	if opens != 1 {
		t.Fatalf("inspection opened package %d times, want exactly once", opens)
	}
}

func TestPackageInspectionResultDoesNotExposePayloadsOrRawDocuments(t *testing.T) {
	typeOfResult := reflect.TypeOf(PackageInspectionResult{})
	for _, forbidden := range []string{
		"Payloads",
		"Signature",
		"PublicKey",
		"IntegrityData",
	} {
		if _, present := typeOfResult.FieldByName(forbidden); present {
			t.Fatalf("PackageInspectionResult exposes forbidden field %q", forbidden)
		}
	}
}

func TestAlphaPackageReadLimitsRemainRuntimeCompatible(t *testing.T) {
	if got, want := AlphaRuntimePackageReadLimits(), AlphaPackageReadLimits(); !reflect.DeepEqual(got, want) {
		t.Fatalf("runtime limits %#v differ from neutral limits %#v", got, want)
	}
}

func assertInspectionSignatureState(
	t *testing.T,
	result PackageInspectionResult,
	state PackageSignatureState,
	declaredKeyID string,
	verifiedSignerKeyID string,
) {
	t.Helper()
	if result.SignatureState != state ||
		result.DeclaredKeyID != declaredKeyID ||
		result.VerifiedSignerKeyID != verifiedSignerKeyID {
		t.Fatalf(
			"signature evidence = (%q, %q, %q), want (%q, %q, %q)",
			result.SignatureState,
			result.DeclaredKeyID,
			result.VerifiedSignerKeyID,
			state,
			declaredKeyID,
			verifiedSignerKeyID,
		)
	}
}

func inspectionTestSignerAndVerifier(
	t *testing.T,
	keyID string,
) (*Ed25519Signer, *Ed25519Verifier) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewEd25519Signer(keyID, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	trustStore := NewTrustStore()
	if err := trustStore.Register(keyID, publicKey); err != nil {
		t.Fatal(err)
	}
	verifier, err := NewEd25519VerifierWithTrustStore(trustStore)
	if err != nil {
		t.Fatal(err)
	}
	return signer, verifier
}

func inspectionTestV2Bundle(targetOS string, targetArch string) ArtifactBundle {
	return ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Runtime: &RuntimeDescriptor{
			Kind: RuntimeKindApplicationExecutable,
			Entrypoint: RuntimeEntrypoint{
				Module:  "app",
				Version: "v1",
			},
			TargetOS:   targetOS,
			TargetArch: targetArch,
		},
		Artifacts: []Artifact{{
			Module:     "app",
			Version:    "v1",
			ImportPath: "example.com/demo/app",
		}},
	}
}

func TestPackageSignatureStateHasExactlyThreeSuccessfulValues(t *testing.T) {
	states := []PackageSignatureState{
		PackageSignatureUnsigned,
		PackageSignatureSignedUnverified,
		PackageSignatureSignedTrusted,
	}
	seen := make(map[PackageSignatureState]struct{}, len(states))
	for _, state := range states {
		if strings.TrimSpace(string(state)) == "" {
			t.Fatal("signature state value is empty")
		}
		seen[state] = struct{}{}
	}
	if len(seen) != 3 {
		t.Fatalf("signature state values are not distinct: %#v", states)
	}
}
