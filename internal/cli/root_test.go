package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()

	commands := map[string]bool{}

	for _, command := range cmd.Commands() {
		commands[command.Name()] = true
	}

	expected := []string{
		"build",
		"build-runnable",
		"config",
		"doctor",
		"run",
		"validate",
		"version",
	}

	for _, name := range expected {
		if !commands[name] {
			t.Fatalf(
				"expected %q command to be registered",
				name,
			)
		}
	}
}

func TestNewRootCommandHelp(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Forge Workspace") {
		t.Fatalf("expected help output, got %q", out.String())
	}
	if !strings.Contains(out.String(), "build-runnable") {
		t.Fatalf("expected build-runnable in root help, got %q", out.String())
	}
	if !strings.Contains(out.String(), "run") {
		t.Fatalf("expected run in root help, got %q", out.String())
	}
	if !strings.Contains(out.String(), "validate") {
		t.Fatalf("expected validate in root help, got %q", out.String())
	}
}

func TestRunCommandHelpExposesOnlySupportedFlags(t *testing.T) {
	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"run", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	help := out.String()
	for _, required := range []string{"run <package.zip>", "--trusted-key", "--key-id", "no sandbox"} {
		if !strings.Contains(help, required) {
			t.Fatalf("run help lacks %q: %q", required, help)
		}
	}
	for _, unsupported := range []string{"--output", "--args", "--env", "--cwd", "--target", "--force", "--unsigned", "--manifest"} {
		if strings.Contains(help, unsupported) {
			t.Fatalf("run help exposes unsupported flag %q: %q", unsupported, help)
		}
	}
}

func TestNewRootCommandVersionFlags(t *testing.T) {
	old := AppVersion
	AppVersion = "1.2.3"
	defer func() { AppVersion = old }()

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Version: 1.2.3") {
		t.Fatalf("expected version output, got %q", out.String())
	}
}

func TestNewRootCommandVersionSubcommand(t *testing.T) {
	old := AppVersion
	AppVersion = "2.0.0"
	defer func() { AppVersion = old }()

	cmd := NewRootCommand()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out.String(), "Version : 2.0.0") {
		t.Fatalf("expected version command output, got %q", out.String())
	}
}

func TestNewRootCommandUnknownCommand(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"unknown"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected unknown command error")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got %q", err.Error())
	}
}

func TestNewRootCommandDoesNotDuplicateCommands(t *testing.T) {
	left := NewRootCommand()
	right := NewRootCommand()

	if len(left.Commands()) != len(right.Commands()) {
		t.Fatalf("expected repeated root commands to have same subcommand count, got %d and %d", len(left.Commands()), len(right.Commands()))
	}

	seen := make(map[string]bool, len(left.Commands()))
	for _, sub := range left.Commands() {
		if seen[sub.Name()] {
			t.Fatalf("duplicate command name %q in root command", sub.Name())
		}
		seen[sub.Name()] = true
	}
}
