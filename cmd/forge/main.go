package main

import (
	"os"

	"github.com/kaizenforyou91/forge/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
