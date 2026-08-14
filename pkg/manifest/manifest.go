package manifest

// Manifest defines the declarative application manifest.
//
// The contract is intentionally small in the foundation phase.
// The manifest contract is intentionally small.
// Loading, structural validation, and module resolution are provided
// by the Manifest Engine foundation.
type Manifest struct {
	Version string   `yaml:"version" json:"version"`
	Name    string   `yaml:"name" json:"name"`
	Modules []Module `yaml:"modules" json:"modules"`
}

// Module describes a manifest-declared application module.
type Module struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}
