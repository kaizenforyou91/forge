package config

import "testing"

func TestEncryptDecryptConfig(t *testing.T) {

	cfg := Default()

	cfg.Secrets.APIKey = "my-secret-key"
	cfg.Secrets.Token = "my-token"

	if err := EncryptConfig(&cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Secrets.APIKey == "my-secret-key" {
		t.Fatal("apikey not encrypted")
	}

	if err := DecryptConfig(&cfg); err != nil {
		t.Fatal(err)
	}

	if cfg.Secrets.APIKey != "my-secret-key" {
		t.Fatal("decrypt failed")
	}

	if cfg.Secrets.Token != "my-token" {
		t.Fatal("decrypt failed")
	}
}
