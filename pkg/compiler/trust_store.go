package compiler

import (
	"crypto/ed25519"
	"sort"
	"strings"
	"sync"
)

// TrustedKey represents one trusted package signing key.
type TrustedKey struct {
	KeyID     string
	PublicKey ed25519.PublicKey
}

// TrustStore stores trusted Ed25519 public keys.
//
// The store is safe for concurrent access.
// Public keys are copied on input and output to prevent aliasing.
type TrustStore struct {
	mu   sync.RWMutex
	keys map[string]ed25519.PublicKey
}

// NewTrustStore creates an empty trust store.
func NewTrustStore() *TrustStore {
	return &TrustStore{
		keys: make(map[string]ed25519.PublicKey),
	}
}

// Register adds a trusted public key.
//
// Register rejects:
//   - empty key IDs
//   - invalid public key lengths
//   - duplicate key IDs
func (s *TrustStore) Register(
	keyID string,
	publicKey ed25519.PublicKey,
) error {
	if s == nil {
		return ErrNilTrustStore
	}

	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return ErrInvalidTrustKey
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return ErrInvalidTrustKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[keyID]; exists {
		return ErrDuplicateTrustKey
	}

	s.keys[keyID] = append(
		ed25519.PublicKey(nil),
		publicKey...,
	)

	return nil
}

// Get returns a trusted key by ID.
//
// The returned public key is an independent copy.
func (s *TrustStore) Get(
	keyID string,
) (TrustedKey, error) {
	if s == nil {
		return TrustedKey{}, ErrNilTrustStore
	}

	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return TrustedKey{}, ErrInvalidTrustKey
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	publicKey, ok := s.keys[keyID]
	if !ok {
		return TrustedKey{}, ErrTrustKeyNotFound
	}

	return TrustedKey{
		KeyID: keyID,
		PublicKey: append(
			ed25519.PublicKey(nil),
			publicKey...,
		),
	}, nil
}

// Has reports whether a trusted key exists.
func (s *TrustStore) Has(keyID string) bool {
	if s == nil {
		return false
	}

	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.keys[keyID]
	return ok
}

// Remove removes a trusted key.
//
// Removing an unknown key is reported as ErrTrustKeyNotFound.
func (s *TrustStore) Remove(keyID string) error {
	if s == nil {
		return ErrNilTrustStore
	}

	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return ErrInvalidTrustKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.keys[keyID]; !ok {
		return ErrTrustKeyNotFound
	}

	delete(s.keys, keyID)
	return nil
}

// List returns trusted keys in deterministic key-ID order.
//
// Returned public keys are independent copies.
func (s *TrustStore) List() []TrustedKey {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	keyIDs := make([]string, 0, len(s.keys))

	for keyID := range s.keys {
		keyIDs = append(keyIDs, keyID)
	}

	sort.Strings(keyIDs)

	result := make([]TrustedKey, 0, len(keyIDs))

	for _, keyID := range keyIDs {
		result = append(result, TrustedKey{
			KeyID: keyID,
			PublicKey: append(
				ed25519.PublicKey(nil),
				s.keys[keyID]...,
			),
		})
	}

	return result
}

// Snapshot returns an immutable-by-convention copy of the store contents.
//
// The returned slice and public keys do not alias the store.
func (s *TrustStore) Snapshot() []TrustedKey {
	return s.List()
}
