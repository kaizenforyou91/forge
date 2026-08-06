package config

import "testing"

func TestValidateSchema(t *testing.T) {

	cfg := Default()

	if err := ValidateSchema(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSchemaFail(t *testing.T) {

	cfg := Default()

	cfg.Project.Name = ""

	if err := ValidateSchema(cfg); err == nil {
		t.Fatal("expected validation error")
	}
}
