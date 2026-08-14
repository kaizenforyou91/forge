package compiler

import (
	"context"
)

// Command represents one external process execution request.
type Command struct {
	Name string
	Args []string
}

// CommandResult represents captured command execution output.
type CommandResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// CommandRunner abstracts operating-system process execution.
type CommandRunner interface {
	Run(context.Context, Command) (CommandResult, error)
}
