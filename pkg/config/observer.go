package config

type Observer interface {
	OnConfigReload(cfg Config)
}
