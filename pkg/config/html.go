package config

import (
	"html"
	"strings"
)

// GenerateHTML generates HTML documentation.
func GenerateHTML(cfg Config) string {

	md := GenerateMarkdown(cfg)

	var b strings.Builder

	b.WriteString("<!DOCTYPE html>")
	b.WriteString("<html><head>")
	b.WriteString("<meta charset=\"UTF-8\">")
	b.WriteString("<title>Forge Configuration</title>")
	b.WriteString("</head><body>")
	b.WriteString("<pre>")
	b.WriteString(html.EscapeString(md))
	b.WriteString("</pre>")
	b.WriteString("</body></html>")

	return b.String()
}
