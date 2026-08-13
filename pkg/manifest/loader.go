package manifest

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// LoadYAML loads a Forge manifest from a YAML file.
func LoadYAML(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest

	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("load manifest %q: %w", path, err)
	}

	return manifest, nil
}
