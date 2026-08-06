package config

import "fmt"

func ValidateSchema(cfg Config) error {

	if err := cfg.Validate(); err != nil {
		return err
	}

	_, err := GenerateJSONSchema(cfg)
	if err != nil {
		return fmt.Errorf("generate schema: %w", err)
	}

	return nil
}
