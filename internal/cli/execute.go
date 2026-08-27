package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

// Execute executes the Forge CLI root command.
func Execute() error {
	return executeWithInterruptContext(
		NewRootCommand(),
		func(parent context.Context) (context.Context, context.CancelFunc) {
			return signal.NotifyContext(parent, os.Interrupt)
		},
	)
}

type executeContextFactory func(context.Context) (context.Context, context.CancelFunc)

func executeWithInterruptContext(
	root *cobra.Command,
	contextFactory executeContextFactory,
) error {
	if root == nil {
		return fmt.Errorf("root command is nil")
	}
	if contextFactory == nil {
		return fmt.Errorf("execute context factory is nil")
	}

	ctx, stop := contextFactory(root.Context())
	if ctx == nil || stop == nil {
		if stop != nil {
			stop()
		}
		return fmt.Errorf("execute context factory returned an incomplete lifecycle")
	}
	defer stop()

	return root.ExecuteContext(ctx)
}
