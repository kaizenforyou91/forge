package config

import (
	"fmt"
	"reflect"
	"strings"
)

// GenerateMarkdown generates Markdown documentation.
func GenerateMarkdown(cfg Config) string {

	var b strings.Builder

	b.WriteString("# Forge Configuration\n\n")

	buildMarkdown(reflect.ValueOf(cfg), "", &b)

	return b.String()
}

func buildMarkdown(v reflect.Value, prefix string, b *strings.Builder) {

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {

		field := t.Field(i)

		name := field.Tag.Get("yaml")
		if name == "" {
			name = field.Name
		}

		full := name
		if prefix != "" {
			full = prefix + "." + name
		}

		if field.Type.Kind() == reflect.Struct {

			buildMarkdown(v.Field(i), full, b)
			continue
		}

		b.WriteString(
			fmt.Sprintf(
				"- `%s` (%s) = `%v`\n",
				full,
				field.Type.Name(),
				v.Field(i).Interface(),
			),
		)
	}
}
