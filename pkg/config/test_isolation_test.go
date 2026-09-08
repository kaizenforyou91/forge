package config

import "testing"

func isolateConfigTest(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("FORGE_CONFIG_KEY", "")
}
