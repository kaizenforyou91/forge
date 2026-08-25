package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cwd-error="+err.Error())
		os.Exit(2)
	}
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "stdin-error="+err.Error())
		os.Exit(3)
	}

	fmt.Println("fixture=process-success")
	fmt.Println("cwd=" + workingDirectory)
	fmt.Printf("args=%d\n", len(os.Args))
	fmt.Printf("stdin-bytes=%d\n", len(stdin))
	fmt.Println("secret=" + os.Getenv("FORGE_RUNNER_SECRET_TEST"))
	fmt.Println("path=" + os.Getenv("PATH"))
	fmt.Println("home=" + os.Getenv("HOME"))
	fmt.Println("tmpdir=" + os.Getenv("TMPDIR"))
	fmt.Println("userprofile=" + os.Getenv("USERPROFILE"))
	fmt.Println("temp=" + os.Getenv("TEMP"))
	fmt.Println("tmp=" + os.Getenv("TMP"))

	fmt.Fprintln(os.Stderr, "fixture=process-success-stderr")
}
