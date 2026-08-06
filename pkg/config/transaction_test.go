package config

import (
	"os"
	"testing"
)

func TestTransaction(t *testing.T) {

	path := "transaction_test.yaml"

	defer os.Remove(path)
	defer os.Remove(path + ".bak")

	cfg := Config{
		Project: Project{
			Name:    "forge",
			Version: "1.0.0",
		},
		Runtime: Runtime{
			Environment: "development",
			LogLevel:    "info",
		},
		Server: Server{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Plugins: Plugins{
			Logger: LoggerPlugin{
				Enabled: true,
			},
		},
	}

	tx := NewTransaction(path)

	if err := tx.Commit(cfg); err != nil {
		t.Fatal(err)
	}
}
