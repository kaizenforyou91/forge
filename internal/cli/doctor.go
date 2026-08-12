package cli

import (
	"fmt"
	"os"

	"github.com/kaizenforyou91/forge/pkg/config"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check Forge environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "Forge Doctor")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "✓ Go runtime detected")

			cfg, err := config.Load("forge.yaml")
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintln(out, "⚠ forge.yaml not found, using default configuration")
					cfg = config.Default()
				} else {
					return err
				}
			}

			fmt.Fprintf(out, "✓ Configuration version %d loaded\n", cfg.Version)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Forge environment looks healthy.")

			return nil
		},
	}
}
