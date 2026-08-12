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

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Forge version",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "Forge CLI")
			fmt.Fprintf(out, "Version : %s\n", AppVersion)
			fmt.Fprintf(out, "Commit  : %s\n", Commit)
			fmt.Fprintf(out, "Built   : %s\n", BuildTime)

			return nil
		},
	}
}
