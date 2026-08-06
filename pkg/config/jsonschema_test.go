package config

import (
	"testing"
)

func TestGenerateJSONSchema(t *testing.T) {

	data, err := GenerateJSONSchema(Default())
	if err != nil {
		t.Fatal(err)
	}

	if len(data) == 0 {
		t.Fatal("empty schema")
	}

	if string(data) == "{}" {
		t.Fatal("invalid schema")
	}
}
