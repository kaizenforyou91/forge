package compiler

import (
	"context"
	"reflect"
	"testing"
)

type fakeCommandRunner struct {
	commands []Command
	result   CommandResult
	err      error
}

func (r *fakeCommandRunner) Run(
	ctx context.Context,
	command Command,
) (CommandResult, error) {
	r.commands = append(r.commands, Command{
		Name: command.Name,
		Args: append([]string(nil), command.Args...),
	})

	return r.result, r.err
}

func TestFakeCommandRunnerContract(t *testing.T) {
	var _ CommandRunner = (*fakeCommandRunner)(nil)
}

func TestCommandRunnerReceivesCommand(t *testing.T) {
	runner := &fakeCommandRunner{
		result: CommandResult{
			ExitCode: 0,
		},
	}

	_, err := runner.Run(
		context.Background(),
		Command{
			Name: "go",
			Args: []string{"build"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	want := []Command{
		{
			Name: "go",
			Args: []string{"build"},
		},
	}

	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf(
			"unexpected commands:\nwant %#v\ngot  %#v",
			want,
			runner.commands,
		)
	}
}
