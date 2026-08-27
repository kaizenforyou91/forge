package main

import (
	"log"
	"os"

	"github.com/kaizenforyou91/forge/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Print(err)
		os.Exit(cli.ExitCode(err))
	}
}
