package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCommand(t *testing.T) {
	cmd := NewRootCommand()

	if cmd == nil {
		t.Fatal("expected root command")
	}

	subcommands := cmd.Commands()
	if len(subcommands) != 3 {
		t.Fatalf("expected 3 subcommands, got %d", len(subcommands))
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
