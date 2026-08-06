package config

import "testing"

func TestMerge(t *testing.T) {

	base := Default()

	override := Config{}

	override.Server.Port = 9090
	override.Runtime.LogLevel = "debug"

	Merge(&base, override)

	if base.Server.Port != 9090 {
		t.Fatal("port not merged")
	}

	if base.Runtime.LogLevel != "debug" {
		t.Fatal("log level not merged")
	}

	if base.Server.Host != "localhost" {
		t.Fatal("unexpected host overwrite")
	}
}
