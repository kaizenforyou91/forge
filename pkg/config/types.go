package config

// Project contains project metadata.
type Project struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// Runtime defines runtime configuration.
type Runtime struct {
	Environment string `yaml:"environment"`
	LogLevel    string `yaml:"log_level"`
}

// Server defines server settings.
type Server struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LoggerPlugin configures the logger plugin.
type LoggerPlugin struct {
	Enabled bool `yaml:"enabled"`
}

// Secrets stores encrypted configuration values.
type Secrets struct {
	APIKey string `yaml:"api_key"`
	Token  string `yaml:"token"`
}

// Plugins contains plugin configurations.
type Plugins struct {
	Logger LoggerPlugin `yaml:"logger"`
}

// Config represents the root Forge configuration.
type Config struct {
	Version int `yaml:"version"`

	Project Project `yaml:"project"`
	Runtime Runtime `yaml:"runtime"`
	Server  Server  `yaml:"server"`

	Secrets Secrets `yaml:"secrets"`

	Plugins Plugins `yaml:"plugins"`
}
