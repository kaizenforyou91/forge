package config

// Default returns the default Forge configuration.
func Default() Config {

	return Config{

		Version: CurrentVersion,

		Project: Project{
			Name:    "forge-app",
			Version: "0.1.0",
		},

		Runtime: Runtime{
			Environment: "development",
			LogLevel:    "info",
		},

		Server: Server{
			Host: "localhost",
			Port: 8080,
		},

		Secrets: Secrets{},

		Plugins: Plugins{
			Logger: LoggerPlugin{
				Enabled: true,
			},
		},
	}
}
