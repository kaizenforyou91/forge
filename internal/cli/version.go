package cli

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Build   = "unknown"
)

func Version() {
	fmt.Println("Forge CLI")
	fmt.Printf("Version : %s\n", Version)
	fmt.Printf("Commit  : %s\n", Commit)
	fmt.Printf("Built   : %s\n", Build)
}
