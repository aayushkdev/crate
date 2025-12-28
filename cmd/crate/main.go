package main

import (
	"fmt"
	"os"

	"github.com/aayushkdev/crate/internal/cli"
	"github.com/aayushkdev/crate/internal/exec"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "init" {
		exec.InitContainer(os.Args[2:])
		return
	}

	err := cli.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "crate:", err)
		os.Exit(1)
	}
}
