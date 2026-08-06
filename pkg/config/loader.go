package config

import (
	"os"

	"go.yaml.in/yaml/v3"
)

// Load loads a configuration file.
func Load(path string) (Config, error) {

	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	if err := DecryptConfig(&cfg); err != nil {
		return cfg, err
	}

	// Before Load Hooks
	if err := RunBeforeLoad(&cfg); err != nil {
		return cfg, err
	}

	// Environment Override
	if err := cfg.OverrideFromEnv(); err != nil {
		return cfg, err
	}

	// Migration
	if err := Migrate(&cfg); err != nil {
		return cfg, err
	}

	// Validation Hooks
	if err := RunBeforeValidate(&cfg); err != nil {
		return cfg, err
	}

	// Validate Config
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	if err := RunAfterValidate(&cfg); err != nil {
		return cfg, err
	}

	// After Load Hooks
	if err := RunAfterLoad(&cfg); err != nil {
		return cfg, err
	}

	SetCache(cfg)

	return cfg, nil
}
