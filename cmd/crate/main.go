package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/aayushkdev/crate/internal/runtime"
)

func main() {
	if len(os.Args) >= 3 {
		containerID := os.Args[2]
		command := os.Args[3:]

		switch os.Args[1] {
		case "init":
			runtime.InitContainer(containerID, command)
			return
		case "exec-init":
			runtime.ExecInit(containerID, command)
			return
		case "exec-child":
			runtime.ExecChild(containerID, command)
			return
		}
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
