package main

import (
	"log"

	"github.com/kaizenforyou91/forge/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Fatal(err)
	}
}
