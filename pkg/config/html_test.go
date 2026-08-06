package config

import (
	"strings"
	"testing"
)

func TestGenerateHTML(t *testing.T) {

	html := GenerateHTML(Default())

	if !strings.Contains(html, "<html") {
		t.Fatal("invalid html")
	}

	if !strings.Contains(html, "Forge Configuration") {
		t.Fatal("missing title")
	}
}
