package config

type Project struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type Runtime struct {
	Environment string `yaml:"environment"`
	LogLevel    string `yaml:"log_level"`
}

type Server struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type LoggerPlugin struct {
	Enabled bool `yaml:"enabled"`
}

type Plugins struct {
	Logger LoggerPlugin `yaml:"logger"`
}

type Config struct {
	Project Project `yaml:"project"`
	Runtime Runtime `yaml:"runtime"`
	Server  Server  `yaml:"server"`
	Plugins Plugins `yaml:"plugins"`
}