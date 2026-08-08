package config

import (
	"os"
	"sync"
)

// Store represents a runtime configuration key-value store.
type Store struct {
	mu     sync.RWMutex
	values map[string]string
}

// NewStore creates a new configuration store.
func NewStore() *Store {
	return &Store{
		values: make(map[string]string),
	}
}

// Set stores a configuration value.
func (s *Store) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.values[key] = value
}

// Get returns a configuration value.
//
// Environment variables take precedence over values stored
// in the configuration store.
func (s *Store) Get(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.values[key]
}

// Has reports whether a configuration key exists.
func (s *Store) Has(key string) bool {
	if _, ok := os.LookupEnv(key); ok {
		return true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.values[key]

	return ok
}
