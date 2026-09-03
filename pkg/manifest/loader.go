package manifest

import (
	"fmt"
	"os"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

// LoadYAML loads a Forge manifest from a YAML file.
func LoadYAML(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest

	if err := decodeStrictManifestYAML(data, &manifest); err != nil {
		return Manifest{}, forgeerrors.Wrap(
			err,
			forgeerrors.CodeInvalidManifest,
			fmt.Sprintf("load YAML manifest %q", path),
		)
	}

	return manifest, nil
}
