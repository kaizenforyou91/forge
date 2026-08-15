package compiler

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestGenerateEd25519Signer(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	if signer == nil {
		t.Fatal("expected signer")
	}

	signature, err := signer.Sign([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}

	if signature.KeyID != "forge-dev" {
		t.Fatalf(
			"expected key ID %q, got %q",
			"forge-dev",
			signature.KeyID,
		)
	}

	if err := signature.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestEd25519SignerProducesValidSignature(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("forge package integrity")

	signature, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewEd25519Verifier()

	publicKey, err := base64.StdEncoding.DecodeString(
		signature.PublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifier.TrustKey(
		signature.KeyID,
		ed25519.PublicKey(publicKey),
	); err != nil {
		t.Fatal(err)
	}

	if err := verifier.Verify(
		payload,
		signature,
	); err != nil {
		t.Fatal(err)
	}
}

func TestEd25519SignerRejectsInvalidKey(t *testing.T) {
	_, err := NewEd25519Signer(
		"forge-dev",
		ed25519.PrivateKey([]byte("invalid")),
	)

	if err == nil {
		t.Fatal("expected invalid key error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}

func TestEd25519SignerRejectsEmptyKeyID(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewEd25519Signer("", privateKey)
	if err == nil {
		t.Fatal("expected empty key ID error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}

func TestPackageSignatureValidate(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	if err := signature.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestPackageSignatureValidateRejectsVersion(t *testing.T) {
	signature := PackageSignature{
		Version:   999,
		Algorithm: packageSignatureAlgorithm,
		KeyID:     "forge-dev",
		PublicKey: base64.StdEncoding.EncodeToString(
			make([]byte, ed25519.PublicKeySize),
		),
		Signature: base64.StdEncoding.EncodeToString(
			make([]byte, ed25519.SignatureSize),
		),
	}

	err := signature.Validate()
	if err == nil {
		t.Fatal("expected invalid version error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}

func TestPackageSignatureValidateRejectsAlgorithm(t *testing.T) {
	signature := PackageSignature{
		Version:   packageSignatureVersion,
		Algorithm: "sha256",
		KeyID:     "forge-dev",
		PublicKey: base64.StdEncoding.EncodeToString(
			make([]byte, ed25519.PublicKeySize),
		),
		Signature: base64.StdEncoding.EncodeToString(
			make([]byte, ed25519.SignatureSize),
		),
	}

	err := signature.Validate()
	if err == nil {
		t.Fatal("expected invalid algorithm error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}

func TestPackageSignatureValidateRejectsInvalidPublicKey(t *testing.T) {
	signature := PackageSignature{
		Version:   packageSignatureVersion,
		Algorithm: packageSignatureAlgorithm,
		KeyID:     "forge-dev",
		PublicKey: "invalid",
		Signature: base64.StdEncoding.EncodeToString(
			make([]byte, ed25519.SignatureSize),
		),
	}

	err := signature.Validate()
	if err == nil {
		t.Fatal("expected invalid public key error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}

func TestMarshalPackageSignature(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := MarshalPackageSignature(signature)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatal("expected serialized signature")
	}
}

func TestUnmarshalPackageSignature(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	original, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	data, err := MarshalPackageSignature(original)
	if err != nil {
		t.Fatal(err)
	}

	got, err := UnmarshalPackageSignature(data)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(original, got) {
		t.Fatalf(
			"signature mismatch: expected=%#v got=%#v",
			original,
			got,
		)
	}
}

func TestUnmarshalPackageSignatureRejectsInvalidJSON(t *testing.T) {
	_, err := UnmarshalPackageSignature(
		[]byte(`{"version":`),
	)

	if err == nil {
		t.Fatal("expected invalid JSON error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}

func TestUnmarshalPackageSignatureRejectsEmptyDocument(t *testing.T) {
	_, err := UnmarshalPackageSignature([]byte("   "))

	if err == nil {
		t.Fatal("expected empty document error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}

func TestPackageSignatureSerializationIsDeterministic(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	first, err := MarshalPackageSignature(signature)
	if err != nil {
		t.Fatal(err)
	}

	second, err := MarshalPackageSignature(signature)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf(
			"expected deterministic serialization:\n%s\n%s",
			first,
			second,
		)
	}
}

func TestEd25519VerifierRejectsTamperedPayload(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := signer.Sign([]byte("original"))
	if err != nil {
		t.Fatal(err)
	}

	publicKey, err := base64.StdEncoding.DecodeString(
		signature.PublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewEd25519Verifier()

	if err := verifier.TrustKey(
		signature.KeyID,
		ed25519.PublicKey(publicKey),
	); err != nil {
		t.Fatal(err)
	}

	err = verifier.Verify(
		[]byte("tampered"),
		signature,
	)

	if err == nil {
		t.Fatal("expected signature mismatch")
	}

	if !errors.Is(err, ErrSignatureMismatch) {
		t.Fatalf(
			"expected ErrSignatureMismatch, got %v",
			err,
		)
	}
}

func TestEd25519VerifierRejectsUnknownKey(t *testing.T) {
	signer, err := GenerateEd25519Signer("untrusted")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewEd25519Verifier()

	err = verifier.Verify(
		[]byte("payload"),
		signature,
	)

	if err == nil {
		t.Fatal("expected untrusted key error")
	}

	if !errors.Is(err, ErrUntrustedPackageKey) {
		t.Fatalf(
			"expected ErrUntrustedPackageKey, got %v",
			err,
		)
	}
}

func TestEd25519VerifierRejectsWrongTrustedKey(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-prod")
	if err != nil {
		t.Fatal(err)
	}

	otherSigner, err := GenerateEd25519Signer("forge-prod")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	otherSignature, err := otherSigner.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	otherPublicKey, err := base64.StdEncoding.DecodeString(
		otherSignature.PublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	verifier := NewEd25519Verifier()

	if err := verifier.TrustKey(
		"forge-prod",
		ed25519.PublicKey(otherPublicKey),
	); err != nil {
		t.Fatal(err)
	}

	err = verifier.Verify(
		[]byte("payload"),
		signature,
	)

	if err == nil {
		t.Fatal("expected trusted-key mismatch")
	}

	if !errors.Is(err, ErrUntrustedPackageKey) {
		t.Fatalf(
			"expected ErrUntrustedPackageKey, got %v",
			err,
		)
	}
}

func TestEd25519VerifierRejectsMalformedSignature(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	signature.Signature = base64.StdEncoding.EncodeToString(
		[]byte("bad"),
	)

	verifier := NewEd25519Verifier()

	publicKey, err := base64.StdEncoding.DecodeString(
		signature.PublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifier.TrustKey(
		signature.KeyID,
		ed25519.PublicKey(publicKey),
	); err != nil {
		t.Fatal(err)
	}

	err = verifier.Verify(
		[]byte("payload"),
		signature,
	)

	if err == nil {
		t.Fatal("expected malformed signature error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}

func TestPackageSignatureRejectsWhitespaceKeyID(t *testing.T) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	signature, err := signer.Sign([]byte("payload"))
	if err != nil {
		t.Fatal(err)
	}

	signature.KeyID = strings.Repeat(" ", 3)

	err = signature.Validate()

	if err == nil {
		t.Fatal("expected whitespace key ID error")
	}

	if !errors.Is(err, ErrInvalidPackageSignature) {
		t.Fatalf(
			"expected ErrInvalidPackageSignature, got %v",
			err,
		)
	}
}
