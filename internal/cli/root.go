package cli

import (
	"fmt"
)

func Run(args []string) int {
	if len(args) == 0 {
		Help()
		return 0
	}

	switch args[0] {

	case "version":
		Version()

	case "doctor":
		Doctor()

	case "help":
		Help()

	default:
		fmt.Printf("Unknown command: %s\n\n", args[0])
		Help()
		return 1
	}

	return 0
}
