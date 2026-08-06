package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
)

const keyFile = ".forge/config/key"

func LoadKey() ([]byte, error) {

	// 1. Environment
	if v := os.Getenv("FORGE_CONFIG_KEY"); v != "" {

		key, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}

		if len(key) != 32 {
			return nil, errors.New("FORGE_CONFIG_KEY must decode to exactly 32 bytes")
		}

		return key, nil
	}

	// 2. Key File
	if data, err := os.ReadFile(keyFile); err == nil {

		key, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, err
		}

		if len(key) != 32 {
			return nil, errors.New("invalid key length")
		}

		return key, nil
	}

	// 3. Generate New
	return GenerateKey()
}

func GenerateKey() ([]byte, error) {

	key := make([]byte, 32)

	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(keyFile), 0755); err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(key)

	if err := os.WriteFile(keyFile, []byte(encoded), 0600); err != nil {
		return nil, err
	}

	return key, nil
}
