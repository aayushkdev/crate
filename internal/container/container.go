package container

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/aayushkdev/crate/internal/fs"
)

func InitContainer(args []string) {

	Fatal(syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""))

	Fatal(syscall.Sethostname([]byte("crate")))

	Fatal(fs.Setup("rootfs/alpinefs"))

	cmd := exec.Command(args[0], args[1:]...)
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
