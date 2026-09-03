package manifest

import (
	"fmt"
	"os"

	forgeerrors "github.com/kaizenforyou91/forge/pkg/errors"
)

// LoadJSON loads a Forge manifest from a JSON file.
func LoadJSON(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest

	if err := decodeStrictManifestJSON(data, &manifest); err != nil {
		return Manifest{}, forgeerrors.Wrap(
			err,
			forgeerrors.CodeInvalidManifest,
			fmt.Sprintf("load JSON manifest %q", path),
		)
	}

	return manifest, nil
}
