package config

import (
	"strings"
	"testing"
)

func TestGenerateMarkdown(t *testing.T) {

	md := GenerateMarkdown(Default())

	if !strings.Contains(md, "project.name") {
		t.Fatal("missing project.name")
	}

	if !strings.Contains(md, "runtime.environment") {
		t.Fatal("missing runtime.environment")
	}

	if !strings.Contains(md, "server.port") {
		t.Fatal("missing server.port")
	}
}
