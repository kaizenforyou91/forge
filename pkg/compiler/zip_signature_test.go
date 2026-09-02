package compiler

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"path/filepath"
	"testing"
)

func TestZIPPackagerWithSignerIncludesSignature(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "signed.zip")

	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	packager := NewZIPPackagerWithSigner(signer)

	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
		},
	}

	payloads := map[string][]byte{
		"http@v1": []byte("http-artifact"),
	}

	if err := packager.Package(
		bundle,
		payloads,
		output,
	); err != nil {
		t.Fatal(err)
	}

	reader, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	found := false
	foundIntegrityV2 := false

	for _, file := range reader.File {
		switch file.Name {
		case signatureManifestPath:
			found = true
			signature, err := UnmarshalPackageSignature(readArchiveFile(t, file))
			if err != nil {
				t.Fatal(err)
			}
			if signature.Version != 1 {
				t.Fatalf("expected signature version 1, got %d", signature.Version)
			}
		case integrityManifestPath:
			integrity, err := UnmarshalPackageIntegrity(readArchiveFile(t, file))
			if err != nil {
				t.Fatal(err)
			}
			foundIntegrityV2 = integrity.Version == 2
		}
	}

	if !found {
		t.Fatal("expected signature.json in signed package")
	}
	if !foundIntegrityV2 {
		t.Fatal("expected signed integrity schema version 2")
	}
}

func TestSignedZIPPackagePreservesExactUnicodeSignerKeyID(t *testing.T) {
	const keyID = "kunci-\u00e9-e\u0301"
	dir := t.TempDir()
	path := filepath.Join(dir, "unicode-signer.zip")
	signer, err := GenerateEd25519Signer(keyID)
	if err != nil {
		t.Fatal(err)
	}
	store := NewTrustStore()
	if err := store.Register(keyID, signer.public); err != nil {
		t.Fatal(err)
	}
	verifier, err := NewEd25519VerifierWithTrustStore(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		path,
	); err != nil {
		t.Fatal(err)
	}

	result, err := NewZIPPackageReaderWithVerifier(verifier).ReadDetailed(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.VerifiedSignerKeyID != keyID {
		t.Fatalf("verified signer KeyID = %q, want exact %q", result.VerifiedSignerKeyID, keyID)
	}
}

func TestZIPPackageReaderWithVerifierAcceptsValidSignature(
	t *testing.T,
) {
	dir := t.TempDir()
	output := filepath.Join(dir, "signed.zip")

	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	signatureProbe, err := signer.Sign([]byte("probe"))
	if err != nil {
		t.Fatal(err)
	}

	publicKey, err := base64.StdEncoding.DecodeString(
		signatureProbe.PublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewEd25519Verifier()

	if err := verifier.TrustKey(
		"forge-dev",
		ed25519.PublicKey(publicKey),
	); err != nil {
		t.Fatal(err)
	}

	bundle := ArtifactBundle{
		ManifestName:    "demo",
		ManifestVersion: "v1",
		Artifacts: []Artifact{
			{
				Module:     "http",
				Version:    "v1",
				ImportPath: "github.com/kaizenforyou91/forge/pkg/http",
			},
		},
	}

	payloads := map[string][]byte{
		"http@v1": []byte("http-artifact"),
	}

	if err := NewZIPPackagerWithSigner(signer).Package(
		bundle,
		payloads,
		output,
	); err != nil {
		t.Fatal(err)
	}

	gotBundle, gotPayloads, err :=
		NewZIPPackageReaderWithVerifier(verifier).Read(output)

	if err != nil {
		t.Fatal(err)
	}

	if gotBundle.ManifestName != "demo" {
		t.Fatalf(
			"expected demo, got %q",
			gotBundle.ManifestName,
		)
	}

	if gotBundle.Artifacts[0].ImportPath !=
		"github.com/kaizenforyou91/forge/pkg/http" {
		t.Fatalf(
			"unexpected artifact import path: %q",
			gotBundle.Artifacts[0].ImportPath,
		)
	}

	if string(gotPayloads["http@v1"]) != "http-artifact" {
		t.Fatalf("unexpected artifact payload")
	}
}

func TestZIPPackageReaderWithVerifierRejectsTamperedSignature(
	t *testing.T,
) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	tamperedPath := filepath.Join(dir, "tampered.zip")

	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}
	otherSigner, err := GenerateEd25519Signer("other")
	if err != nil {
		t.Fatal(err)
	}
	probe, err := signer.Sign([]byte("probe"))
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(probe.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewEd25519Verifier()
	if err := verifier.TrustKey(probe.KeyID, ed25519.PublicKey(publicKey)); err != nil {
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
	otherSignature, err := otherSigner.Sign(entries[integrityManifestPath])
	if err != nil {
		t.Fatal(err)
	}
	entries[signatureManifestPath], err = MarshalPackageSignature(otherSignature)
	if err != nil {
		t.Fatal(err)
	}
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err = NewZIPPackageReaderWithVerifier(verifier).Read(tamperedPath)
	if !errors.Is(err, ErrUntrustedPackageKey) {
		t.Fatalf("expected ErrUntrustedPackageKey, got %v", err)
	}
}

func TestSignedZIPPackageRejectsPackageMetadataTampering(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	tamperedPath := filepath.Join(dir, "tampered.zip")

	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}
	probe, err := signer.Sign([]byte("probe"))
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(probe.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewEd25519Verifier()
	if err := verifier.TrustKey(probe.KeyID, ed25519.PublicKey(publicKey)); err != nil {
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
	entries[packageMetadataPath] = append(entries[packageMetadataPath], ' ')
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err = NewZIPPackageReaderWithVerifier(verifier).Read(tamperedPath)
	if !errors.Is(err, ErrIntegrityMismatch) {
		t.Fatalf("expected ErrIntegrityMismatch, got %v", err)
	}
}

func TestSignedZIPPackageRejectsRegeneratedIntegrityWithExistingSignature(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	tamperedPath := filepath.Join(dir, "tampered.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)

	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, validPath)
	entries[packageMetadataPath] = append(entries[packageMetadataPath], '\n')
	integrity, err := BuildPackageIntegrity(
		entries[packageMetadataPath],
		testPackageBundle(),
		entries[bundleManifestPath],
		testPackagePayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}
	entries[integrityManifestPath], err = MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err = NewZIPPackageReaderWithVerifier(verifier).Read(tamperedPath)
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got %v", err)
	}
}

func TestUnsignedZIPPackageAcceptsRegeneratedIntegrity(t *testing.T) {
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
	entries[packageMetadataPath] = append(entries[packageMetadataPath], '\n')
	integrity, err := BuildPackageIntegrity(
		entries[packageMetadataPath],
		testPackageBundle(),
		entries[bundleManifestPath],
		testPackagePayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}
	entries[integrityManifestPath], err = MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}
	writeZIPEntriesForTest(t, changedPath, entries)

	if _, _, err := NewZIPPackageReader().Read(changedPath); err != nil {
		t.Fatalf("expected internally consistent unsigned package, got %v", err)
	}
}

func TestSignedZIPPackageRejectsModifiedFormatDeclaration(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	tamperedPath := filepath.Join(dir, "tampered.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, validPath)
	entries[packageMetadataPath] = []byte(
		`{"package_format_version":0,"bundle_schema_version":1}`,
	)
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err := NewZIPPackageReaderWithVerifier(verifier).Read(tamperedPath)
	if !errors.Is(err, ErrUnsupportedPackageFormat) {
		t.Fatalf("expected ErrUnsupportedPackageFormat, got %v", err)
	}
}

func TestSignedZIPPackageRejectsSignatureByteTampering(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	tamperedPath := filepath.Join(dir, "tampered.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, validPath)
	signature, err := UnmarshalPackageSignature(entries[signatureManifestPath])
	if err != nil {
		t.Fatal(err)
	}
	signature.Signature = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	entries[signatureManifestPath], err = MarshalPackageSignature(signature)
	if err != nil {
		t.Fatal(err)
	}
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err = NewZIPPackageReaderWithVerifier(verifier).Read(tamperedPath)
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got %v", err)
	}
}

func TestSignedZIPPackageRejectsInvalidSignatureKeyIDBeforeTrustRouting(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid.zip")
	tamperedPath := filepath.Join(dir, "invalid-key-id.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)
	if err := NewZIPPackagerWithSigner(signer).Package(
		testPackageBundle(),
		testPackagePayloads(),
		validPath,
	); err != nil {
		t.Fatal(err)
	}

	entries := readZIPEntriesForTest(t, validPath)
	original := entries[signatureManifestPath]
	changed := bytes.Replace(
		original,
		[]byte(`"key_id":"forge-dev"`),
		[]byte(`"key_id":"forge\u001bdev"`),
		1,
	)
	if bytes.Equal(changed, original) {
		t.Fatal("signature fixture did not contain expected key ID")
	}
	entries[signatureManifestPath] = changed
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err := NewZIPPackageReaderWithVerifier(verifier).Read(tamperedPath)
	if !errors.Is(err, ErrInvalidPackageSignature) || !errors.Is(err, ErrInvalidKeyID) {
		t.Fatalf("expected signature and KeyID errors, got %v", err)
	}
	if errors.Is(err, ErrUntrustedPackageKey) {
		t.Fatalf("invalid signature KeyID reached trust routing: %v", err)
	}
}

func TestSignedZIPPackageV2UsesSignatureSchemaV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "signed-v2.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)
	writeTestPackageV2(t, NewZIPPackagerWithSigner(signer), path)
	entries := readZIPEntriesForTest(t, path)
	signature, err := UnmarshalPackageSignature(entries[signatureManifestPath])
	if err != nil {
		t.Fatal(err)
	}
	if signature.Version != 1 {
		t.Fatalf("expected signature schema version 1, got %d", signature.Version)
	}
	reader, err := NewZIPPackageReaderWithPolicyAndVerifier(
		StrictPackageVerificationPolicy(), verifier,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reader.Read(path); err != nil {
		t.Fatal(err)
	}
}

func TestSignedZIPPackageV2RejectsRegeneratedIntegrityWithExistingSignature(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid-v2.zip")
	tamperedPath := filepath.Join(dir, "tampered-v2.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)
	writeTestPackageV2(t, NewZIPPackagerWithSigner(signer), validPath)
	entries := readZIPEntriesForTest(t, validPath)
	entries[bundleManifestPath] = bytes.Replace(
		entries[bundleManifestPath],
		[]byte(`"target_arch":"amd64"`),
		[]byte(`"target_arch":"arm64"`),
		1,
	)
	bundle, err := unmarshalArtifactBundleForSchema(
		entries[bundleManifestPath], artifactBundleSchemaVersionV2,
	)
	if err != nil {
		t.Fatal(err)
	}
	integrity, err := buildPackageIntegrityForSchema(
		artifactBundleSchemaVersionV2,
		entries[packageMetadataPath],
		bundle,
		entries[bundleManifestPath],
		testRunnablePackagePlaceholderPayloads(),
	)
	if err != nil {
		t.Fatal(err)
	}
	entries[integrityManifestPath], err = MarshalPackageIntegrity(integrity)
	if err != nil {
		t.Fatal(err)
	}
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err = NewZIPPackageReaderWithVerifier(verifier).Read(tamperedPath)
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got %v", err)
	}
}

func TestSignedZIPPackageV2RejectsIntegrityByteTampering(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid-v2.zip")
	tamperedPath := filepath.Join(dir, "tampered-v2.zip")
	signer, verifier := trustedTestSignerAndVerifier(t)
	writeTestPackageV2(t, NewZIPPackagerWithSigner(signer), validPath)
	entries := readZIPEntriesForTest(t, validPath)
	entries[integrityManifestPath] = append(entries[integrityManifestPath], ' ')
	writeZIPEntriesForTest(t, tamperedPath, entries)

	_, _, err := NewZIPPackageReaderWithVerifier(verifier).Read(tamperedPath)
	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf("expected ErrSignatureMismatch, got %v", err)
	}
}

func TestSignedZIPPackageV2RejectsUntrustedSigner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "untrusted-v2.zip")
	signer, err := GenerateEd25519Signer("untrusted")
	if err != nil {
		t.Fatal(err)
	}
	writeTestPackageV2(t, NewZIPPackagerWithSigner(signer), path)
	_, _, err = NewZIPPackageReaderWithVerifier(NewEd25519Verifier()).Read(path)
	if !errors.Is(err, ErrUntrustedPackageKey) {
		t.Fatalf("expected ErrUntrustedPackageKey, got %v", err)
	}
}

func trustedTestSignerAndVerifier(
	t *testing.T,
) (*Ed25519Signer, *Ed25519Verifier) {
	t.Helper()

	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}
	probe, err := signer.Sign([]byte("probe"))
	if err != nil {
		t.Fatal(err)
	}
	publicKey, err := base64.StdEncoding.DecodeString(probe.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	verifier := NewEd25519Verifier()
	if err := verifier.TrustKey(probe.KeyID, ed25519.PublicKey(publicKey)); err != nil {
		t.Fatal(err)
	}

	return signer, verifier
}
