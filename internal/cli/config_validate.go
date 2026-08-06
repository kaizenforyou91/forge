package cli

import "github.com/spf13/cobra"

func newConfigValidateCommand() *cobra.Command {

	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Run: func(cmd *cobra.Command, args []string) {

			cmd.Println("config validate")

		},
	}

}
