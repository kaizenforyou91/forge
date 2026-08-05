package config

func Default() Config {

	return Config{

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

		Plugins: Plugins{
			Logger: LoggerPlugin{
				Enabled: true,
			},
		},
	}
}
