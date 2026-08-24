package compiler

import (
	"crypto/ed25519"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNewZIPPackageReaderUsesDefaultPolicy(t *testing.T) {
	reader := NewZIPPackageReader()

	if reader == nil {
		t.Fatal("expected reader")
	}

	if !reader.policy.RequireIntegrity {
		t.Fatal("default reader must require integrity")
	}

	if reader.policy.RequireSignature {
		t.Fatal("default reader must not require signature")
	}
}

func TestNewZIPPackageReaderWithVerifierUsesDefaultPolicy(
	t *testing.T,
) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewEd25519Verifier()

	publicKey, err := decodeSignaturePublicKey(
		mustSignProbe(t, signer),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifier.TrustKey(
		"forge-dev",
		ed25519.PublicKey(publicKey),
	); err != nil {
		t.Fatal(err)
	}

	reader := NewZIPPackageReaderWithVerifier(verifier)

	if reader == nil {
		t.Fatal("expected reader")
	}

	if !reader.policy.RequireIntegrity {
		t.Fatal("default reader must require integrity")
	}

	if reader.policy.RequireSignature {
		t.Fatal("default reader must keep signature optional")
	}
}

func TestNewZIPPackageReaderWithPolicyAllowsDefaultPolicy(
	t *testing.T,
) {
	reader, err := NewZIPPackageReaderWithPolicy(
		DefaultPackageVerificationPolicy(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if reader == nil {
		t.Fatal("expected reader")
	}
}

func TestNewZIPPackageReaderWithPolicyAllowsStrictPolicy(
	t *testing.T,
) {
	reader, err := NewZIPPackageReaderWithPolicy(
		StrictPackageVerificationPolicy(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if reader == nil {
		t.Fatal("expected reader")
	}

	if !reader.policy.RequireIntegrity {
		t.Fatal("strict policy must require integrity")
	}

	if !reader.policy.RequireSignature {
		t.Fatal("strict policy must require signature")
	}
}

func TestNewZIPPackageReaderWithPolicyAndVerifier(
	t *testing.T,
) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewEd25519Verifier()

	signature := mustSignProbe(t, signer)

	publicKey, err := decodeSignaturePublicKey(signature)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifier.TrustKey(
		"forge-dev",
		ed25519.PublicKey(publicKey),
	); err != nil {
		t.Fatal(err)
	}

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}

	if reader == nil {
		t.Fatal("expected reader")
	}

	if !reader.policy.RequireIntegrity {
		t.Fatal("strict policy must require integrity")
	}

	if !reader.policy.RequireSignature {
		t.Fatal("strict policy must require signature")
	}

	if reader.verifier == nil {
		t.Fatal("expected verifier")
	}
}

func TestNewZIPPackageReaderWithPolicyAndVerifierRejectsNilVerifier(
	t *testing.T,
) {
	_, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		nil,
	)

	if !errors.Is(err, ErrPackageVerifierRequired) {
		t.Fatalf(
			"expected ErrPackageVerifierRequired, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderStrictPolicyAcceptsSignedPackage(
	t *testing.T,
) {
	dir := t.TempDir()
	path := filepath.Join(dir, "signed.zip")

	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewEd25519Verifier()

	signature := mustSignProbe(t, signer)

	publicKey, err := decodeSignaturePublicKey(signature)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifier.TrustKey(
		"forge-dev",
		ed25519.PublicKey(publicKey),
	); err != nil {
		t.Fatal(err)
	}

	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	if err := NewZIPPackagerWithSigner(signer).Package(
		bundle,
		payloads,
		path,
	); err != nil {
		t.Fatal(err)
	}

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}

	gotBundle, gotPayloads, err := reader.Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if gotBundle.ManifestName != bundle.ManifestName {
		t.Fatalf(
			"expected manifest name %q, got %q",
			bundle.ManifestName,
			gotBundle.ManifestName,
		)
	}

	expectedPayloads := testPackagePayloads()

	if string(gotPayloads["http@v1"]) != string(expectedPayloads["http@v1"]) {
		t.Fatal("unexpected HTTP artifact payload")
	}
}

func TestZIPPackageReaderStrictPolicyRejectsUnsignedPackage(
	t *testing.T,
) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unsigned.zip")

	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	if err := NewZIPPackager().Package(
		bundle,
		payloads,
		path,
	); err != nil {
		t.Fatal(err)
	}

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		NewEd25519Verifier(),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = reader.Read(path)

	if !errors.Is(err, ErrMissingPackageSignature) {
		t.Fatalf(
			"expected ErrMissingPackageSignature, got %v",
			err,
		)
	}
}

func TestZIPPackageReaderStrictPolicyRejectsMissingIntegrity(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	changedPath := filepath.Join(dir, "missing-integrity.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, validPath)
	delete(entries, integrityManifestPath)
	writeZIPEntriesForTest(t, changedPath, entries)

	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(),
		verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = reader.Read(changedPath)
	if !errors.Is(err, ErrMissingPackageIntegrity) {
		t.Fatalf("expected ErrMissingPackageIntegrity, got %v", err)
	}
}

func TestZIPPackageReaderDefaultPolicyAcceptsUnsignedPackage(
	t *testing.T,
) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unsigned.zip")

	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	if err := NewZIPPackager().Package(
		bundle,
		payloads,
		path,
	); err != nil {
		t.Fatal(err)
	}

	gotBundle, gotPayloads, err :=
		NewZIPPackageReader().Read(path)

	if err != nil {
		t.Fatal(err)
	}

	if gotBundle.ManifestName != bundle.ManifestName {
		t.Fatalf(
			"expected manifest %q, got %q",
			bundle.ManifestName,
			gotBundle.ManifestName,
		)
	}

	expectedPayloads := testPackagePayloads()

	if string(gotPayloads["http@v1"]) != string(expectedPayloads["http@v1"]) {
		t.Fatal("unexpected payload")
	}
}

func TestZIPPackageReaderDefaultPolicyRejectsMissingIntegrity(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	changedPath := filepath.Join(dir, "missing-integrity.zip")
	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, validPath)
	delete(entries, integrityManifestPath)
	writeZIPEntriesForTest(t, changedPath, entries)

	_, _, err := NewZIPPackageReader().Read(changedPath)
	if !errors.Is(err, ErrMissingPackageIntegrity) {
		t.Fatalf("expected ErrMissingPackageIntegrity, got %v", err)
	}
}

func TestZIPPackageReaderRejectsSignatureWithoutIntegrity(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	changedPath := filepath.Join(dir, "signature-without-integrity.zip")
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, validPath)
	delete(entries, integrityManifestPath)
	writeZIPEntriesForTest(t, changedPath, entries)

	reader, err := NewZIPPackageReaderWithPolicy(PackageVerificationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = reader.Read(changedPath)
	if !errors.Is(err, ErrMissingPackageIntegrity) {
		t.Fatalf("expected ErrMissingPackageIntegrity, got %v", err)
	}
}

func mustSignProbe(
	t *testing.T,
	signer PackageSigner,
) PackageSignature {
	t.Helper()

	signature, err := signer.Sign([]byte("forge-policy-probe"))
	if err != nil {
		t.Fatal(err)
	}

	return signature
}

func TestZIPPackageReaderWithoutIntegrityAcceptsPackage(
	t *testing.T,
) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unsigned-no-integrity.zip")

	bundle := testPackageBundle()
	payloads := testPackagePayloads()

	packageData := map[string][]byte{}
	packageData[packageMetadataPath] = mustPackageMetadataJSON(t)

	bundleData, err := MarshalArtifactBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}

	packageData[bundleManifestPath] = bundleData

	for key, payload := range payloads {
		parts := strings.SplitN(key, "@", 2)

		archivePath := filepath.ToSlash(filepath.Join(
			artifactRootPath,
			parts[0],
			parts[1],
			artifactFileName,
		))

		packageData[archivePath] = append([]byte(nil), payload...)
	}

	writeZIPEntriesForTest(t, path, packageData)

	policy := PackageVerificationPolicy{
		RequireIntegrity: false,
		RequireSignature: false,
	}

	reader, err := NewZIPPackageReaderWithPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}

	gotBundle, gotPayloads, err := reader.Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(gotBundle, bundle) {
		t.Fatalf(
			"expected bundle %#v, got %#v",
			bundle,
			gotBundle,
		)
	}

	if !reflect.DeepEqual(gotPayloads, payloads) {
		t.Fatalf(
			"expected payloads %#v, got %#v",
			payloads,
			gotPayloads,
		)
	}
}

func TestZIPPackageReaderWithoutIntegrityStillValidatesPackageMetadata(
	t *testing.T,
) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid-metadata.zip")

	bundleData, err := MarshalArtifactBundle(testPackageBundle())
	if err != nil {
		t.Fatal(err)
	}

	packageData := map[string][]byte{
		packageMetadataPath: []byte(`{"package_format_version":2,"bundle_schema_version":1}`),
		bundleManifestPath:  bundleData,
	}
	for key, payload := range testPackagePayloads() {
		parts := strings.SplitN(key, "@", 2)
		packageData[filepath.ToSlash(filepath.Join(
			artifactRootPath,
			parts[0],
			parts[1],
			artifactFileName,
		))] = append([]byte(nil), payload...)
	}
	writeZIPEntriesForTest(t, path, packageData)

	reader, err := NewZIPPackageReaderWithPolicy(PackageVerificationPolicy{})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = reader.Read(path)
	if !errors.Is(err, ErrUnsupportedPackageFormat) {
		t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
	}
}

func TestZIPPackageReaderWithoutIntegrityEnforcesStructuralValidation(t *testing.T) {
	policy := PackageVerificationPolicy{}
	reader, err := NewZIPPackageReaderWithPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(map[string][]byte)
		want   error
	}{
		{
			name: "missing package metadata",
			mutate: func(entries map[string][]byte) {
				delete(entries, packageMetadataPath)
			},
			want: ErrLegacyPackageUnsupported,
		},
		{
			name: "malformed package metadata",
			mutate: func(entries map[string][]byte) {
				entries[packageMetadataPath] = []byte(`{"package_format_version":`)
			},
			want: ErrInvalidPackageMetadata,
		},
		{
			name: "invalid bundle",
			mutate: func(entries map[string][]byte) {
				entries[bundleManifestPath] = []byte(`{"manifest_name":"","manifest_version":"v1","artifacts":[]}`)
			},
			want: ErrInvalidArtifactBundle,
		},
		{
			name: "missing payload",
			mutate: func(entries map[string][]byte) {
				delete(entries, "artifacts/http/v1/artifact")
			},
			want: ErrMissingArtifactPayload,
		},
		{
			name: "extra payload",
			mutate: func(entries map[string][]byte) {
				entries["artifacts/extra/v1/artifact"] = []byte("extra")
			},
			want: ErrInvalidArtifactPackage,
		},
		{
			name: "unexpected top-level entry",
			mutate: func(entries map[string][]byte) {
				entries["unexpected.json"] = []byte(`{}`)
			},
			want: ErrInvalidArtifactPackage,
		},
		{
			name: "unsafe entry",
			mutate: func(entries map[string][]byte) {
				entries["../escape"] = []byte("unsafe")
			},
			want: ErrInvalidArtifactPackage,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			validPath := filepath.Join(dir, "valid.zip")
			changedPath := filepath.Join(dir, "changed.zip")
			if err := NewZIPPackager().Package(
				testPackageBundle(),
				testPackagePayloads(),
				validPath,
			); err != nil {
				t.Fatal(err)
			}

			entries := readZIPEntriesForTest(t, validPath)
			delete(entries, integrityManifestPath)
			test.mutate(entries)
			writeZIPEntriesForTest(t, changedPath, entries)

			_, _, err := reader.Read(changedPath)
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestZIPPackageReaderWithoutIntegrityAcceptsSupportedMetadataTamper(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	changedPath := filepath.Join(dir, "changed.zip")
	if err := NewZIPPackager().Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, validPath)
	delete(entries, integrityManifestPath)
	entries[packageMetadataPath] = append(entries[packageMetadataPath], '\n')
	writeZIPEntriesForTest(t, changedPath, entries)

	reader, err := NewZIPPackageReaderWithPolicy(PackageVerificationPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reader.Read(changedPath); err != nil {
		t.Fatalf("expected structurally valid metadata tamper to be accepted, got %v", err)
	}
}
