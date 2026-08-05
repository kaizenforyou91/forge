package cli

import "fmt"

func Help() {

	fmt.Println("Forge CLI")
	fmt.Println()

	fmt.Println("Usage:")

	fmt.Println("  forge version")
	fmt.Println("  forge doctor")
	fmt.Println("  forge help")
}
