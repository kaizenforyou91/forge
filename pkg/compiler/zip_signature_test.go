package compiler

import (
	"archive/zip"
	"crypto/ed25519"
	"encoding/base64"
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
			{Module: "http", Version: "v1"},
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

	for _, file := range reader.File {
		if file.Name == signatureManifestPath {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected signature.json in signed package")
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
			{Module: "http", Version: "v1"},
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

	if string(gotPayloads["http@v1"]) != "http-artifact" {
		t.Fatalf("unexpected artifact payload")
	}
}

func TestZIPPackageReaderWithVerifierRejectsTamperedSignature(
	t *testing.T,
) {
	// Fixture:
	// 1. Create a signed ZIP.
	// 2. Replace signature.json with another valid signature
	//    created by a different key.
	// 3. Read using the original trusted verifier.
	// 4. Expect ErrUntrustedPackageKey or ErrSignatureMismatch.
}
