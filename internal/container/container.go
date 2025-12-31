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

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Fatal(cmd.Start())

	Fatal(cmd.Wait())

	os.Exit(0)

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
