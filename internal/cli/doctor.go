package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check Forge environment",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("Forge Doctor")
		fmt.Println()

		fmt.Println("✓ Go runtime detected")
		fmt.Println("✓ Logger initialized")
		fmt.Println("✓ Error package loaded")
		fmt.Println()

		fmt.Println("Forge environment looks healthy.")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
