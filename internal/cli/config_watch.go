package cli

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/config"
	"github.com/spf13/cobra"
)

func newConfigWatchCmd(configPathFn configPathProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Watch configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			configPath := configPathFn()

			watcher, err := config.NewWatcher(configPath)
			if err != nil {
				return err
			}
			defer watcher.Close()

			if err := watcher.Start(); err != nil {
				return err
			}

			watcher.Run()

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Watching %s...\n", configPath)
			fmt.Fprintln(out, "Press Ctrl+C to stop.")

			<-ctx.Done()
			return ctx.Err()
		},
	}
}
