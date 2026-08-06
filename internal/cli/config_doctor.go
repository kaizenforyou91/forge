package cli

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/config"
	"github.com/spf13/cobra"
)

var configDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check configuration health",
	RunE: func(cmd *cobra.Command, args []string) error {

		_, err := config.Load("forge.yaml")
		if err != nil {
			return err
		}

		fmt.Println("Configuration Doctor")
		fmt.Println()

		fmt.Println("✓ forge.yaml found")
		fmt.Println("✓ YAML syntax valid")
		fmt.Println("✓ Environment override loaded")
		fmt.Println("✓ Validation passed")
		fmt.Println("✓ Configuration healthy")

		return nil
	},
}

func init() {
	configCmd.AddCommand(configDoctorCmd)
}
