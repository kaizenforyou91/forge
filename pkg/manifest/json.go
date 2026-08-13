package manifest

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadJSON loads a Forge manifest from a JSON file.
func LoadJSON(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest

	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("load manifest %q: %w", path, err)
	}

	return manifest, nil
}
