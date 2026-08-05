package config

import (
	"fmt"
)

var (
	validEnvironments = map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
	}

	validLogLevels = map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
)

// Validate validates the configuration.
func (c Config) Validate() error {

	if c.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}

	if c.Project.Version == "" {
		return fmt.Errorf("project.version is required")
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	if !validEnvironments[c.Runtime.Environment] {
		return fmt.Errorf(
			"runtime.environment must be one of development, staging, production",
		)
	}

	if !validLogLevels[c.Runtime.LogLevel] {
		return fmt.Errorf(
			"runtime.log_level must be one of debug, info, warn, error",
		)
	}

	return nil
}
