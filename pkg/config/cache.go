package config

import "sync"

var (
	cache  Config
	loaded bool
	mu     sync.RWMutex
)

func GetCache() (Config, bool) {

	mu.RLock()
	defer mu.RUnlock()

	return cache, loaded
}

func SetCache(cfg Config) {

	mu.Lock()
	defer mu.Unlock()

	cache = cfg
	loaded = true
}

func ClearCache() {

	mu.Lock()
	defer mu.Unlock()

	loaded = false
	cache = Config{}
}
