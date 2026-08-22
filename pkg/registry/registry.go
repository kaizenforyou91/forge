package registry

import (
	"errors"
	"sync"
)

var (
	ErrInvalidPackage   = errors.New("invalid package")
	ErrDuplicatePackage = errors.New("package already registered")
	ErrPackageNotFound  = errors.New("package not found")
)

// Registry stores uniquely identifiable Forge packages.
//
// Package identity is defined by the combination of name and version.
// The registry preserves registration order and is safe for concurrent access.
type Registry struct {
	mu       sync.RWMutex
	packages map[string]Package
	order    []string
}

// New creates an empty package registry.
func New() *Registry {
	return &Registry{
		packages: make(map[string]Package),
		order:    make([]string, 0),
	}
}

// Register adds a package to the registry.
//
// A package must have both a name and a version.
// The same name/version pair cannot be registered twice.
func (r *Registry) Register(pkg Package) error {
	if pkg.Name == "" || pkg.Version == "" {
		return ErrInvalidPackage
	}

	key := packageKey(pkg.Name, pkg.Version)

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.packages[key]; exists {
		return ErrDuplicatePackage
	}

	r.packages[key] = pkg
	r.order = append(r.order, key)

	return nil
}

// EnsureAll registers every missing package in a valid batch.
//
// Existing package identities and repeated identities within the batch are
// accepted as idempotent no-ops. New packages retain their first-occurrence
// input order.
func (r *Registry) EnsureAll(packages []Package) error {
	if r == nil {
		return ErrInvalidPackage
	}

	unique := make([]Package, 0, len(packages))
	seen := make(map[string]struct{}, len(packages))

	for _, pkg := range packages {
		if pkg.Name == "" || pkg.Version == "" {
			return ErrInvalidPackage
		}

		key := packageKey(pkg.Name, pkg.Version)
		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		unique = append(unique, pkg)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	missing := make([]Package, 0, len(unique))
	for _, pkg := range unique {
		key := packageKey(pkg.Name, pkg.Version)
		if _, exists := r.packages[key]; !exists {
			missing = append(missing, pkg)
		}
	}

	for _, pkg := range missing {
		key := packageKey(pkg.Name, pkg.Version)
		r.packages[key] = pkg
		r.order = append(r.order, key)
	}

	return nil
}

// Get returns a package by exact name and version.
func (r *Registry) Get(name, version string) (Package, error) {
	key := packageKey(name, version)

	r.mu.RLock()
	defer r.mu.RUnlock()

	pkg, ok := r.packages[key]
	if !ok {
		return Package{}, ErrPackageNotFound
	}

	return pkg, nil
}

// List returns registered packages in registration order.
//
// The returned slice is an independent snapshot.
func (r *Registry) List() []Package {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Package, 0, len(r.order))

	for _, key := range r.order {
		result = append(result, r.packages[key])
	}

	return result
}

// Count returns the number of registered packages.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.packages)
}

func packageKey(name, version string) string {
	return name + "@" + version
}
