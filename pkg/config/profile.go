package config

import (
	"fmt"
	"os"
)

// LoadProfile loads a named configuration profile.
func ProfilePath(base, profile string) string {

	if profile == "" {
		return base + ".yaml"
	}

	return fmt.Sprintf("%s.%s.yaml", base, profile)
}

func LoadProfile(base, profile string) (Config, error) {

	path := ProfilePath(base, profile)

	if _, err := os.Stat(path); err != nil {
		return Config{}, err
	}

	return Load(path)
}

func SaveProfile(base, profile string, cfg Config) error {

	path := ProfilePath(base, profile)

	return Save(path, cfg)
}
