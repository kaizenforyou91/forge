package cli

import (
	"os"

	"github.com/spf13/cobra"
)

func newConfigInitCmd(configPathFn configPathProvider) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Generate default forge.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := configPathFn()

			const yaml = `project:
  name: Forge
  version: dev

runtime:
  environment: development
  log_level: info

server:
  host: 127.0.0.1
  port: 8080

plugins:
  logger:
    enabled: true
`

			return os.WriteFile(configPath, []byte(yaml), 0644)
		},
	}
}
