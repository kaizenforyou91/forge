package cli

import "github.com/spf13/cobra"

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

func init() {
	rootCmd.AddCommand(configCmd)
}
