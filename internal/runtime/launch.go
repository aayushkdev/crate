package runtime

import (
	"fmt"
	"os"
	"os/exec"
)

func LaunchContainer(args []string) error {
	cmd := exec.Command(
		"/proc/self/exe",
		append([]string{"init"}, args...)...,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to launch container: %w", err)
	}

	return nil
}
