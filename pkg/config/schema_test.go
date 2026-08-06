package config

import "testing"

func TestGenerateSchema(t *testing.T) {

	schema := GenerateSchema(Default())

	if schema == nil {
		t.Fatal("schema nil")
	}

	project := schema["project"].(map[string]any)

	if project["name"] != "string" {
		t.Fatal("invalid project.name")
	}

	server := schema["server"].(map[string]any)

	if server["port"] != "int" {
		t.Fatal("invalid server.port")
	}
}
