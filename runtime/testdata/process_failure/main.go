package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("fixture=process-failure")
	fmt.Fprintln(os.Stderr, "fixture=process-failure-stderr")
	os.Exit(23)
}
