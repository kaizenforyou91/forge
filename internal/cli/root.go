package cli

import "github.com/spf13/cobra"

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "forge",
		Short:         "Forge Workspace",
		Long:          "Forge Workspace\n\nModern Development Platform built with Go.",
		Version:       AppVersion,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetVersionTemplate("Forge CLI\nVersion: {{.Version}}\n")

	cmd.AddCommand(
		newVersionCmd(),
		newDoctorCmd(),
		newConfigCmd(),
	)

	return cmd
}
