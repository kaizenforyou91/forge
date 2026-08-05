package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	AppVersion = "dev"
	Commit     = "none"
	BuildTime  = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Forge version",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("Forge CLI")
		fmt.Printf("Version : %s\n", AppVersion)
		fmt.Printf("Commit  : %s\n", Commit)
		fmt.Printf("Built   : %s\n", BuildTime)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
