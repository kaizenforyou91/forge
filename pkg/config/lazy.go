package config

func LoadCached(path string) (Config, error) {

	if cfg, ok := GetCache(); ok {
		return cfg, nil
	}

	cfg, err := Load(path)
	if err != nil {
		return cfg, err
	}

	SetCache(cfg)

	return cfg, nil
}
