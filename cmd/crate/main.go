package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/aayushkdev/crate/internal/container"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "init" {
		// TODO: Use spec instead of manual parsing
		container.InitContainer(os.Args[2], os.Args[3:])
		return
	}

	if err := Execute(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "crate:", err)
		os.Exit(1)
	}

	os.Exit(0)
}
