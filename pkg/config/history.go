package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const historyDir = ".forge/config/history"

func SaveHistory(cfg Config) error {

	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.yaml",
		time.Now().Format("20060102-150405"))

	path := filepath.Join(historyDir, filename)

	return writeConfig(path, cfg)
}
func ListHistory() ([]string, error) {

	entries, err := os.ReadDir(historyDir)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(entries))

	for _, e := range entries {

		if e.IsDir() {
			continue
		}

		result = append(result, e.Name())
	}

	return result, nil
}
