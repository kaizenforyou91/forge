package cli

import "github.com/spf13/cobra"

type configPathProvider func() string

func newConfigCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:           "config",
		Short:         "Configuration management",
		Long:          "Manage Forge configuration and diagnostics.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", "forge.yaml", "Path to Forge configuration file")

	cmd.AddCommand(
		newConfigShowCmd(func() string { return configPath }),
		newConfigValidateCmd(func() string { return configPath }),
		newConfigWatchCmd(func() string { return configPath }),
		newConfigDoctorCmd(func() string { return configPath }),
		newConfigInitCmd(func() string { return configPath }),
	)

	return cmd
}
