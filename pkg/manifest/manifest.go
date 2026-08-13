package manifest

// Manifest defines the declarative application manifest.
//
// The contract is intentionally small in the foundation phase.
// Parsing, schema validation, and dependency resolution are handled
// by later Manifest Engine milestones.
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
