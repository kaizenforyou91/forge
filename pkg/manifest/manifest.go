package manifest

// Manifest defines the declarative application manifest.
//
// The contract is intentionally small.
// Loading, structural validation, and module resolution are provided
// by the Manifest Engine foundation.
type Manifest struct {
	Version    string                 `yaml:"version" json:"version"`
	Name       string                 `yaml:"name" json:"name"`
	Entrypoint *ApplicationEntrypoint `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Modules    []Module               `yaml:"modules" json:"modules"`
}

// ApplicationEntrypoint identifies the single manifest-declared application
// module selected by a runnable workflow.
type ApplicationEntrypoint struct {
	Module  string `yaml:"module" json:"module"`
	Version string `yaml:"version" json:"version"`
}

// Module describes a manifest-declared application module.
type Module struct {
	Name         string       `yaml:"name" json:"name"`
	Version      string       `yaml:"version" json:"version"`
	ImportPath   string       `yaml:"import_path" json:"import_path"`
	Dependencies []Dependency `yaml:"dependencies" json:"dependencies"`
}

// Dependency describes an exact module dependency.
type Dependency struct {
	Name    string `yaml:"name" json:"name"`
	Version string `yaml:"version" json:"version"`
}
