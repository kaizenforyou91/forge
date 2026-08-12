package cli

import (
	"fmt"
	"os"

	"github.com/kaizenforyou91/forge/pkg/config"
	"github.com/spf13/cobra"
)

func newConfigShowCmd(configPathFn configPathProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			configPath := configPathFn()

			cfg, err := config.Load(configPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(out, "Forge configuration file %q not found. Using default configuration.\n\n", configPath)
					cfg = config.Default()
				} else {
					return err
				}
			}

			fmt.Fprintln(out, "Forge Configuration")
			fmt.Fprintln(out, "===================")
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Configuration file: %s\n\n", configPath)

			fmt.Fprintln(out, "Project")
			fmt.Fprintln(out, "-------")
			fmt.Fprintf(out, "Name       : %s\n", cfg.Project.Name)
			fmt.Fprintf(out, "Version    : %s\n", cfg.Project.Version)
			fmt.Fprintln(out)

			fmt.Fprintln(out, "Runtime")
			fmt.Fprintln(out, "-------")
			fmt.Fprintf(out, "Environment : %s\n", cfg.Runtime.Environment)
			fmt.Fprintf(out, "Log Level   : %s\n", cfg.Runtime.LogLevel)
			fmt.Fprintln(out)

			fmt.Fprintln(out, "Server")
			fmt.Fprintln(out, "------")
			fmt.Fprintf(out, "Host : %s\n", cfg.Server.Host)
			fmt.Fprintf(out, "Port : %d\n", cfg.Server.Port)
			fmt.Fprintln(out)

			fmt.Fprintln(out, "Plugins")
			fmt.Fprintln(out, "-------")
			fmt.Fprintf(out, "Logger : %t\n", cfg.Plugins.Logger.Enabled)

			return nil
		},
	}
}
