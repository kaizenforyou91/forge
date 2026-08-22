package compiler

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrInvalidPackageSource   = errors.New("invalid package source")
	ErrDuplicatePackageSource = errors.New("package source already registered")
	ErrPackageSourceConflict  = errors.New("package source conflict")
	ErrPackageSourceNotFound  = errors.New("package source not found")
)

// PackageSource describes where a registered Forge package is compiled from.
type PackageSource struct {
	Name       string
	Version    string
	ImportPath string
}

// PackageSourceRegistry maps Forge package identity to Go import paths.
type PackageSourceRegistry struct {
	mu      sync.RWMutex
	sources map[string]PackageSource
	order   []string
}

// NewPackageSourceRegistry creates an empty package source registry.
func NewPackageSourceRegistry() *PackageSourceRegistry {
	return &PackageSourceRegistry{
		sources: make(map[string]PackageSource),
		order:   make([]string, 0),
	}
}

// Register adds a package source.
//
// Package identity is defined by name + version.
// The same package identity cannot be registered twice.
func (r *PackageSourceRegistry) Register(
	source PackageSource,
) error {
	if r == nil {
		return ErrInvalidPackageSource
	}

	source, err := normalizePackageSource(source)
	if err != nil {
		return err
	}

	key := packageSourceKey(source.Name, source.Version)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sources[key]; exists {
		return ErrDuplicatePackageSource
	}

	r.sources[key] = source
	r.order = append(r.order, key)

	return nil
}

// Ensure registers a package source or verifies an existing source binding.
//
// Package identity is defined by name + version. Repeated identical source
// bindings are accepted, while a different import path is a conflict.
func (r *PackageSourceRegistry) Ensure(
	source PackageSource,
) error {
	if r == nil {
		return ErrInvalidPackageSource
	}

	source, err := normalizePackageSource(source)
	if err != nil {
		return err
	}

	key := packageSourceKey(source.Name, source.Version)

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.sources[key]
	if exists {
		if existing.ImportPath == source.ImportPath {
			return nil
		}

		return packageSourceConflictError(
			key,
			existing.ImportPath,
			source.ImportPath,
		)
	}

	r.sources[key] = source
	r.order = append(r.order, key)

	return nil
}

// EnsureAll registers or verifies every package source in a batch.
//
// The complete batch is normalized and checked for conflicts before any
// missing source is inserted. New sources retain their first-occurrence input
// order.
func (r *PackageSourceRegistry) EnsureAll(
	sources []PackageSource,
) error {
	if r == nil {
		return ErrInvalidPackageSource
	}

	unique := make([]PackageSource, 0, len(sources))
	batch := make(map[string]PackageSource, len(sources))

	for _, source := range sources {
		normalized, err := normalizePackageSource(source)
		if err != nil {
			return err
		}

		key := packageSourceKey(normalized.Name, normalized.Version)
		existing, exists := batch[key]
		if exists {
			if existing.ImportPath != normalized.ImportPath {
				return packageSourceConflictError(
					key,
					existing.ImportPath,
					normalized.ImportPath,
				)
			}

			continue
		}

		batch[key] = normalized
		unique = append(unique, normalized)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, source := range unique {
		key := packageSourceKey(source.Name, source.Version)
		existing, exists := r.sources[key]
		if exists && existing.ImportPath != source.ImportPath {
			return packageSourceConflictError(
				key,
				existing.ImportPath,
				source.ImportPath,
			)
		}
	}

	for _, source := range unique {
		key := packageSourceKey(source.Name, source.Version)
		if _, exists := r.sources[key]; exists {
			continue
		}

		r.sources[key] = source
		r.order = append(r.order, key)
	}

	return nil
}

// Resolve returns the source for an exact package identity.
func (r *PackageSourceRegistry) Resolve(
	name,
	version string,
) (PackageSource, error) {
	if r == nil {
		return PackageSource{}, ErrPackageSourceNotFound
	}

	key := packageSourceKey(
		strings.TrimSpace(name),
		strings.TrimSpace(version),
	)

	r.mu.RLock()
	defer r.mu.RUnlock()

	source, ok := r.sources[key]
	if !ok {
		return PackageSource{}, ErrPackageSourceNotFound
	}

	return source, nil
}

// List returns package sources in registration order.
//
// The returned slice is an independent snapshot.
func (r *PackageSourceRegistry) List() []PackageSource {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PackageSource, 0, len(r.order))

	for _, key := range r.order {
		result = append(result, r.sources[key])
	}

	return result
}

// Count returns the number of registered package sources.
func (r *PackageSourceRegistry) Count() int {
	if r == nil {
		return 0
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.sources)
}

func packageSourceKey(name, version string) string {
	return name + "@" + version
}

func normalizePackageSource(
	source PackageSource,
) (PackageSource, error) {
	source.Name = strings.TrimSpace(source.Name)
	source.Version = strings.TrimSpace(source.Version)
	source.ImportPath = strings.TrimSpace(source.ImportPath)

	if source.Name == "" ||
		source.Version == "" ||
		source.ImportPath == "" {
		return PackageSource{}, ErrInvalidPackageSource
	}

	return source, nil
}

func packageSourceConflictError(
	key,
	existingImportPath,
	requestedImportPath string,
) error {
	return fmt.Errorf(
		"%w: %s is bound to %q, not %q",
		ErrPackageSourceConflict,
		key,
		existingImportPath,
		requestedImportPath,
	)
}

type PackageSourceResolver interface {
	Resolve(name, version string) (PackageSource, error)
}
