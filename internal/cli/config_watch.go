package cli

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/config"
	"github.com/spf13/cobra"
)

var configWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch configuration",

	RunE: func(cmd *cobra.Command, args []string) error {

		watcher, err := config.NewWatcher("forge.yaml")
		if err != nil {
			return err
		}

		defer watcher.Close()

		if err := watcher.Start(); err != nil {
			return err
		}

		watcher.Run()

		fmt.Println("Watching forge.yaml...")
		fmt.Println("Press Ctrl+C to stop.")

		select {}
	},
}

func init() {
	configCmd.AddCommand(configWatchCmd)
}
