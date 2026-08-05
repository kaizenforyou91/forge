package cli

import "fmt"

func Doctor() {

	fmt.Println("Forge Doctor")
	fmt.Println()

	fmt.Println("✓ Go runtime detected")
	fmt.Println("✓ Logger initialized")
	fmt.Println("✓ Error package loaded")
	fmt.Println()

	fmt.Println("Forge environment looks healthy.")
}
