package cli

import (
	"github.com/kaizenforyou91/forge/internal/bootstrap"
	"github.com/kaizenforyou91/forge/pkg/app"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the default Forge CLI command.
func NewRootCommand() *cobra.Command {
	return NewRootCommandWithApplication(
		bootstrap.NewApplication(),
	)
}

// NewRootCommandWithApplication creates the Forge CLI command
// using the supplied application composition.
func NewRootCommandWithApplication(
	application *app.App,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "forge",
		Short:         "Forge Workspace",
		Long:          "Forge Workspace\n\nModern Development Platform built with Go.",
		Version:       AppVersion,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetVersionTemplate("Forge CLI\nVersion: {{.Version}}\n")
	cmd.SetContext(NewApplicationContext(application))

	cmd.AddCommand(
		newVersionCmd(),
		newDoctorCmd(),
		newConfigCmd(),
		newBuildCmd(),
	)

	return cmd
}
