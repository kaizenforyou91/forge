package cli

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/config"
	"github.com/spf13/cobra"
)

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {

		mgr, err := config.NewManager("forge.yaml")
		if err != nil {
			return err
		}

		cfg := mgr.Config()

		fmt.Println("Forge Configuration")
		fmt.Println("===================")
		fmt.Println()

		fmt.Println("Project")
		fmt.Println("-------")
		fmt.Println("Name       :", cfg.Project.Name)
		fmt.Println("Version    :", cfg.Project.Version)
		fmt.Println()

		fmt.Println("Runtime")
		fmt.Println("-------")
		fmt.Println("Environment :", cfg.Runtime.Environment)
		fmt.Println("Log Level   :", cfg.Runtime.LogLevel)
		fmt.Println()

		fmt.Println("Server")
		fmt.Println("------")
		fmt.Println("Host :", cfg.Server.Host)
		fmt.Println("Port :", cfg.Server.Port)
		fmt.Println()

		fmt.Println("Plugins")
		fmt.Println("-------")
		fmt.Println("Logger :", cfg.Plugins.Logger.Enabled)

		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
}
