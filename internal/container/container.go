package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/aayushkdev/crate/internal/fs"
)

func InitContainer(containerID string, command []string) {
	cfg, err := ReadConfig(containerID)
	Fatal(err)
	rootfs := rootfsDir(containerID)

	Fatal(syscall.Sethostname([]byte("crate")))

	Fatal(fs.Setup(rootfs, cfg.Rootless))

	// Replace PID 1 with user process
	Fatal(syscall.Exec(command[0], command, os.Environ()))
}

func Fatal(err error) {
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "crate: init exec failed", err)
		os.Exit(1)
	}
}
