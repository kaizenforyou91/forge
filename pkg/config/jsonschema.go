package config

import (
	"encoding/json"
	"reflect"
)

// GenerateJSONSchema returns the JSON Schema.
func GenerateJSONSchema(cfg Config) ([]byte, error) {

	root := map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type":    "object",
		"properties": buildJSONProperties(
			reflect.TypeOf(cfg),
		),
	}

	return json.MarshalIndent(root, "", "  ")
}

func buildJSONProperties(t reflect.Type) map[string]any {

	props := make(map[string]any)

	for i := 0; i < t.NumField(); i++ {

		f := t.Field(i)

		name := f.Tag.Get("yaml")
		if name == "" {
			name = f.Name
		}

		if f.Type.Kind() == reflect.Struct {

			props[name] = map[string]any{
				"type": "object",
				"properties": buildJSONProperties(
					f.Type,
				),
			}

			continue
		}

		props[name] = map[string]any{
			"type": jsonType(f.Type.Kind()),
		}
	}

	return props
}

func jsonType(k reflect.Kind) string {

	switch k {

	case reflect.String:
		return "string"

	case reflect.Int,
		reflect.Int32,
		reflect.Int64:
		return "integer"

	case reflect.Bool:
		return "boolean"

	case reflect.Float32,
		reflect.Float64:
		return "number"

	default:
		return "string"
	}
}
