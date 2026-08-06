package config

import "os"

func LoadMerged(base, profile string) (Config, error) {

	baseCfg, err := Load(ProfilePath(base, ""))
	if err != nil {
		return Config{}, err
	}

	if profile == "" {
		return baseCfg, nil
	}

	profilePath := ProfilePath(base, profile)

	if _, err := os.Stat(profilePath); err != nil {
		return baseCfg, nil
	}

	override, err := Load(profilePath)
	if err != nil {
		return Config{}, err
	}

	Merge(&baseCfg, override)

	if err := baseCfg.Validate(); err != nil {
		return Config{}, err
	}

	return baseCfg, nil
}

// Merge merges two configurations.
func Merge(dst *Config, src Config) {

	if src.Project.Name != "" {
		dst.Project.Name = src.Project.Name
	}

	if src.Project.Version != "" {
		dst.Project.Version = src.Project.Version
	}

	if src.Runtime.Environment != "" {
		dst.Runtime.Environment = src.Runtime.Environment
	}

	if src.Runtime.LogLevel != "" {
		dst.Runtime.LogLevel = src.Runtime.LogLevel
	}

	if src.Server.Host != "" {
		dst.Server.Host = src.Server.Host
	}

	if src.Server.Port != 0 {
		dst.Server.Port = src.Server.Port
	}

	dst.Plugins = src.Plugins
	dst.Secrets = src.Secrets
}
