package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// This fixture uses its own executable, never a shell or external program.
// Marker handshakes establish readiness; deadlines only bound broken tests.
func main() {
	if len(os.Args) == 2 && os.Args[1] == "descendant" {
		fmt.Println("descendant=ready")
		fmt.Fprintln(os.Stderr, "descendant-stderr=ready")
		if err := os.WriteFile("descendant.ready", []byte("ready"), 0600); err != nil {
			panic(err)
		}
		time.Sleep(2 * time.Minute) // Failure safety; correct tests kill this scope.
		return
	}
	path, err := os.Executable()
	if err != nil {
		panic(err)
	}
	child := exec.Command(path, "descendant")
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	if err := child.Start(); err != nil {
		panic(err)
	}
	await("descendant.ready")
	fmt.Println("leader=ready")
	if err := os.WriteFile("leader.ready", []byte("ready"), 0600); err != nil {
		panic(err)
	}
	await("leader.exit")
}

func await(path string) {
	deadline := time.Now().Add(60 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			panic("fixture handshake deadline")
		}
		time.Sleep(time.Millisecond)
	}
}
