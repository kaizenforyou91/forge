package config

import (
	"os"

	"go.yaml.in/yaml/v3"
)

func Save(path string, cfg Config) error {

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	tmp := path + ".tmp"

	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}
func Exists(path string) bool {

	_, err := os.Stat(path)

	return err == nil
}