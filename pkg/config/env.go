package config

import (
	"os"
	"strconv"
)

func (c *Config) OverrideFromEnv() error {

	if v := os.Getenv("FORGE_SERVER_HOST"); v != "" {
		c.Server.Host = v
	}

	if v := os.Getenv("FORGE_SERVER_PORT"); v != "" {

		port, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		c.Server.Port = port
	}

	if v := os.Getenv("FORGE_LOGGER_LEVEL"); v != "" {
		c.Runtime.LogLevel = v
	}

	return nil
}
