package cli

import (
	"fmt"

	"github.com/kaizenforyou91/forge/pkg/config"
	"github.com/spf13/cobra"
)

func newConfigValidateCmd(configPathFn configPathProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			configPath := configPathFn()

			if _, err := config.Load(configPath); err != nil {
				return err
			}

			fmt.Fprintln(out, "Configuration valid.")
			return nil
		},
	}
}
