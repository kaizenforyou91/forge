package config

import "fmt"

const CurrentVersion = 1

// CheckVersion validates the configuration version.
func CheckVersion(cfg Config) error {

	if cfg.Version > CurrentVersion {
		return fmt.Errorf(
			"config version %d is newer than supported version %d",
			cfg.Version,
			CurrentVersion,
		)
	}

	return nil
}

// Upgrade upgrades a configuration to the latest version.
func Upgrade(cfg *Config) error {

	if cfg.Version == 0 {
		cfg.Version = 1
	}

	cfg.Version = CurrentVersion

	return nil
}
