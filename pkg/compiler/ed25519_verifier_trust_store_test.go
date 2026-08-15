package compiler

import (
	"crypto/ed25519"
	"errors"
	"testing"
)

func TestNewEd25519VerifierUsesTrustStore(t *testing.T) {
	verifier := NewEd25519Verifier()

	if verifier == nil {
		t.Fatal("expected verifier")
	}

	if verifier.TrustStore() == nil {
		t.Fatal("expected trust store")
	}
}

func TestNewEd25519VerifierWithTrustStore(t *testing.T) {
	store := NewTrustStore()

	verifier, err := NewEd25519VerifierWithTrustStore(store)
	if err != nil {
		t.Fatal(err)
	}

	if verifier.TrustStore() != store {
		t.Fatal("verifier does not use supplied trust store")
	}
}

func TestNewEd25519VerifierWithTrustStoreRejectsNil(
	t *testing.T,
) {
	verifier, err := NewEd25519VerifierWithTrustStore(nil)

	if verifier != nil {
		t.Fatal("expected nil verifier")
	}

	if !errors.Is(err, ErrNilTrustStore) {
		t.Fatalf(
			"expected ErrNilTrustStore, got %v",
			err,
		)
	}
}

func TestEd25519VerifierTrustKeyUsesTrustStore(
	t *testing.T,
) {
	verifier := NewEd25519Verifier()

	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifier.TrustKey(
		"forge-dev",
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	if !verifier.TrustStore().Has("forge-dev") {
		t.Fatal("expected TrustKey to register in trust store")
	}
}

func TestEd25519VerifierTrustKeyRejectsDuplicate(
	t *testing.T,
) {
	verifier := NewEd25519Verifier()

	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifier.TrustKey(
		"forge-dev",
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	err = verifier.TrustKey(
		"forge-dev",
		publicKey,
	)

	if err == nil {
		t.Fatal("expected duplicate trust key error")
	}
}

func TestEd25519VerifierUsesUpdatedTrustStore(
	t *testing.T,
) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	store := NewTrustStore()

	verifier, err := NewEd25519VerifierWithTrustStore(store)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("forge package integrity")

	signature, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}

	if err := verifier.Verify(
		payload,
		signature,
	); !errors.Is(err, ErrUntrustedPackageKey) {
		t.Fatalf(
			"expected ErrUntrustedPackageKey, got %v",
			err,
		)
	}

	publicKey, err := decodeSignaturePublicKey(signature)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Register(
		signature.KeyID,
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	if err := verifier.Verify(
		payload,
		signature,
	); err != nil {
		t.Fatalf(
			"expected signature verification to succeed: %v",
			err,
		)
	}
}

func TestEd25519VerifierStopsTrustAfterKeyRemoval(
	t *testing.T,
) {
	signer, err := GenerateEd25519Signer("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	store := NewTrustStore()

	verifier, err := NewEd25519VerifierWithTrustStore(store)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("forge package integrity")

	signature, err := signer.Sign(payload)
	if err != nil {
		t.Fatal(err)
	}

	publicKey, err := decodeSignaturePublicKey(signature)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Register(
		signature.KeyID,
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	if err := verifier.Verify(
		payload,
		signature,
	); err != nil {
		t.Fatal(err)
	}

	if err := store.Remove(signature.KeyID); err != nil {
		t.Fatal(err)
	}

	err = verifier.Verify(
		payload,
		signature,
	)

	if !errors.Is(err, ErrUntrustedPackageKey) {
		t.Fatalf(
			"expected ErrUntrustedPackageKey after removal, got %v",
			err,
		)
	}
}

func decodeSignaturePublicKey(
	signature PackageSignature,
) (ed25519.PublicKey, error) {
	return decodeEd25519PublicKey(signature.PublicKey)
}
