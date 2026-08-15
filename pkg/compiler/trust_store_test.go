package compiler

import (
	"crypto/ed25519"
	"errors"
	"reflect"
	"testing"
)

func generateTestPublicKey(t *testing.T) ed25519.PublicKey {
	t.Helper()

	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	return publicKey
}

func TestNewTrustStore(t *testing.T) {
	store := NewTrustStore()

	if store == nil {
		t.Fatal("expected trust store")
	}

	if got := store.List(); len(got) != 0 {
		t.Fatalf("expected empty store, got %d keys", len(got))
	}
}

func TestTrustStoreRegister(t *testing.T) {
	store := NewTrustStore()
	publicKey := generateTestPublicKey(t)

	if err := store.Register(
		"forge-dev",
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	if !store.Has("forge-dev") {
		t.Fatal("expected registered key")
	}
}

func TestTrustStoreRegisterCopiesPublicKey(t *testing.T) {
	store := NewTrustStore()
	publicKey := generateTestPublicKey(t)

	original := append(
		ed25519.PublicKey(nil),
		publicKey...,
	)

	if err := store.Register(
		"forge-dev",
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	publicKey[0] ^= 0xff

	got, err := store.Get("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(
		got.PublicKey,
		original,
	) {
		t.Fatal("trust store aliases registered public key")
	}
}

func TestTrustStoreGetCopiesPublicKey(t *testing.T) {
	store := NewTrustStore()
	publicKey := generateTestPublicKey(t)

	if err := store.Register(
		"forge-dev",
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	got.PublicKey[0] ^= 0xff

	again, err := store.Get("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(
		got.PublicKey,
		again.PublicKey,
	) {
		t.Fatal("Get returned aliased public key")
	}
}

func TestTrustStoreRejectsDuplicate(t *testing.T) {
	store := NewTrustStore()

	key1 := generateTestPublicKey(t)
	key2 := generateTestPublicKey(t)

	if err := store.Register(
		"forge-dev",
		key1,
	); err != nil {
		t.Fatal(err)
	}

	err := store.Register(
		"forge-dev",
		key2,
	)

	if !errors.Is(err, ErrDuplicateTrustKey) {
		t.Fatalf(
			"expected ErrDuplicateTrustKey, got %v",
			err,
		)
	}
}

func TestTrustStoreRejectsEmptyKeyID(t *testing.T) {
	store := NewTrustStore()
	publicKey := generateTestPublicKey(t)

	err := store.Register(
		"",
		publicKey,
	)

	if !errors.Is(err, ErrInvalidTrustKey) {
		t.Fatalf(
			"expected ErrInvalidTrustKey, got %v",
			err,
		)
	}
}

func TestTrustStoreRejectsInvalidPublicKey(t *testing.T) {
	store := NewTrustStore()

	err := store.Register(
		"forge-dev",
		make([]byte, 4),
	)

	if !errors.Is(err, ErrInvalidTrustKey) {
		t.Fatalf(
			"expected ErrInvalidTrustKey, got %v",
			err,
		)
	}
}

func TestTrustStoreGetMissingKey(t *testing.T) {
	store := NewTrustStore()

	_, err := store.Get("missing")

	if !errors.Is(err, ErrTrustKeyNotFound) {
		t.Fatalf(
			"expected ErrTrustKeyNotFound, got %v",
			err,
		)
	}
}

func TestTrustStoreRemove(t *testing.T) {
	store := NewTrustStore()
	publicKey := generateTestPublicKey(t)

	if err := store.Register(
		"forge-dev",
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	if err := store.Remove("forge-dev"); err != nil {
		t.Fatal(err)
	}

	if store.Has("forge-dev") {
		t.Fatal("expected key to be removed")
	}
}

func TestTrustStoreRemoveMissingKey(t *testing.T) {
	store := NewTrustStore()

	err := store.Remove("missing")

	if !errors.Is(err, ErrTrustKeyNotFound) {
		t.Fatalf(
			"expected ErrTrustKeyNotFound, got %v",
			err,
		)
	}
}

func TestTrustStoreListIsDeterministic(t *testing.T) {
	store := NewTrustStore()

	for _, keyID := range []string{
		"forge-prod",
		"forge-dev",
		"forge-core",
	} {
		if err := store.Register(
			keyID,
			generateTestPublicKey(t),
		); err != nil {
			t.Fatal(err)
		}
	}

	got := store.List()

	want := []string{
		"forge-core",
		"forge-dev",
		"forge-prod",
	}

	for i, key := range got {
		if key.KeyID != want[i] {
			t.Fatalf(
				"index %d: expected %q, got %q",
				i,
				want[i],
				key.KeyID,
			)
		}
	}
}

func TestTrustStoreSnapshotDoesNotAlias(t *testing.T) {
	store := NewTrustStore()
	publicKey := generateTestPublicKey(t)

	if err := store.Register(
		"forge-dev",
		publicKey,
	); err != nil {
		t.Fatal(err)
	}

	snapshot := store.Snapshot()

	if len(snapshot) != 1 {
		t.Fatalf(
			"expected 1 key, got %d",
			len(snapshot),
		)
	}

	snapshot[0].PublicKey[0] ^= 0xff
	snapshot[0].KeyID = "changed"

	got, err := store.Get("forge-dev")
	if err != nil {
		t.Fatal(err)
	}

	if got.KeyID != "forge-dev" {
		t.Fatal("snapshot aliases trust-store key ID")
	}

	if got.PublicKey[0] == snapshot[0].PublicKey[0] {
		t.Fatal("snapshot aliases trust-store public key")
	}
}

func TestTrustStoreNilReceiver(t *testing.T) {
	var store *TrustStore

	if store.Has("forge-dev") {
		t.Fatal("nil receiver must not report keys")
	}

	if got := store.List(); got != nil {
		t.Fatal("expected nil list for nil receiver")
	}

	if _, err := store.Get("forge-dev"); !errors.Is(
		err,
		ErrNilTrustStore,
	) {
		t.Fatalf(
			"expected ErrNilTrustStore, got %v",
			err,
		)
	}

	if err := store.Register(
		"forge-dev",
		generateTestPublicKey(t),
	); !errors.Is(err, ErrNilTrustStore) {
		t.Fatalf(
			"expected ErrNilTrustStore, got %v",
			err,
		)
	}

	if err := store.Remove(
		"forge-dev",
	); !errors.Is(err, ErrNilTrustStore) {
		t.Fatalf(
			"expected ErrNilTrustStore, got %v",
			err,
		)
	}
}
