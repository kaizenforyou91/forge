package compiler

import (
	"crypto/ed25519"
	"errors"
	"path/filepath"
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

func TestNewZIPPackageReaderWithPolicyRejectsRequiredSignatureWithoutVerifier(
	t *testing.T,
) {
	_, err := NewZIPPackageReaderWithPolicy(
		StrictPackageVerificationPolicy(),
	)

	if !errors.Is(err, ErrPackageVerifierRequired) {
		t.Fatalf(
			"expected ErrPackageVerifierRequired, got %v",
			err,
		)
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
