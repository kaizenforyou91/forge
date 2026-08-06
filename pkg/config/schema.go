package config

import (
	"reflect"
)

// GenerateSchema returns the Forge schema.
func GenerateSchema(cfg Config) map[string]any {
	return buildSchema(reflect.TypeOf(cfg))
}

func buildSchema(t reflect.Type) map[string]any {

	result := make(map[string]any)

	for i := 0; i < t.NumField(); i++ {

		field := t.Field(i)

		name := field.Tag.Get("yaml")
		if name == "" {
			name = field.Name
		}

		if field.Type.Kind() == reflect.Struct {

			result[name] = buildSchema(field.Type)

			continue
		}

		result[name] = field.Type.String()
	}

	return result
}
