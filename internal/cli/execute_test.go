package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/spf13/cobra"
)

func TestExecuteSignalContextPreservesApplicationAndCancellation(t *testing.T) {
	application := app.New()
	root := &cobra.Command{Use: "forge", SilenceUsage: true, SilenceErrors: true}
	root.SetContext(NewApplicationContext(application))
	root.SetArgs([]string{"probe"})
	root.AddCommand(&cobra.Command{
		Use: "probe",
		RunE: func(cmd *cobra.Command, _ []string) error {
			gotApplication, err := ApplicationFromContext(cmd.Context())
			if err != nil {
				return err
			}
			if gotApplication != application {
				t.Fatal("signal-aware context did not preserve the application value")
			}
			if cmd.Context().Err() != context.Canceled {
				t.Fatalf("expected canceled command context, got %v", cmd.Context().Err())
			}
			return cmd.Context().Err()
		},
	})

	stopCalled := false
	err := executeWithInterruptContext(
		root,
		func(parent context.Context) (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(parent)
			cancel()
			return ctx, func() {
				stopCalled = true
				cancel()
			}
		},
	)
	if err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if !stopCalled {
		t.Fatal("expected signal notification stop function to be called")
	}
}

func TestExecuteSignalContextLeavesExistingCommandBehaviorUnchanged(t *testing.T) {
	root := NewRootCommand()
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs([]string{"version"})

	if err := executeWithInterruptContext(
		root,
		func(parent context.Context) (context.Context, context.CancelFunc) {
			return context.WithCancel(parent)
		},
	); err != nil {
		t.Fatal(err)
	}
	if output.Len() == 0 {
		t.Fatal("expected existing version command output")
	}
}

func TestExecuteSignalContextRejectsIncompleteLifecycle(t *testing.T) {
	root := &cobra.Command{Use: "forge"}

	if err := executeWithInterruptContext(nil, context.WithCancel); err == nil {
		t.Fatal("expected nil root error")
	}
	if err := executeWithInterruptContext(root, nil); err == nil {
		t.Fatal("expected nil context factory error")
	}
	if err := executeWithInterruptContext(
		root,
		func(context.Context) (context.Context, context.CancelFunc) {
			return nil, nil
		},
	); err == nil {
		t.Fatal("expected incomplete context lifecycle error")
	}
}
