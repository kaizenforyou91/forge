package config

// Observer receives notifications when the configuration is reloaded.
type Observer interface {
	OnConfigReload(cfg Config)
}
