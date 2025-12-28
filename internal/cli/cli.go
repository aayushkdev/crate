package cli

import (
	"fmt"
	"os"

	"github.com/aayushkdev/crate/internal/runtime"
)

func Execute() error {

	args := os.Args
	if len(args) < 2 {
		return fmt.Errorf("no command specified")
	}

	switch args[1] {
	case "run":
		return run(args[2:])
	default:
		return fmt.Errorf("unknown command: %s", args[1])
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	if err := runtime.LaunchContainer(args); err != nil {
		return err
	}
	return nil
}
