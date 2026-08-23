package compiler

import (
	"archive/zip"
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
