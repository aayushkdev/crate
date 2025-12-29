package container

import (
	"fmt"
	"os"
	"syscall"

	"github.com/aayushkdev/crate/internal/fs"
)

func InitContainer(args []string) {

	Fatal(syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""))

	Fatal(syscall.Sethostname([]byte("crate")))

	Fatal(fs.Setup("rootfs/alpinefs"))

	Fatal(syscall.Exec(args[0], args, os.Environ()))

}

func Fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "crate: init exec failed", err)
		os.Exit(1)
	}
}
