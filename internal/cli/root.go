package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "Forge Workspace",
	Long: `Forge Workspace

Modern Development Platform
Built with Go.`,
}
