package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/aayushkdev/crate/internal/fs"
)

func InitContainer(root bool, image string, command []string) {
	Fatal(syscall.Sethostname([]byte("crate")))

	Fatal(fs.Setup(image, root))

	//TODO: maybe use something like tini for zombie reaping
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
