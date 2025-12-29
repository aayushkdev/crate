package fs

import (
	"os"
	"syscall"
)

func Setup(rootfs string) error {
	if err := syscall.Chroot(rootfs); err != nil {
		return err
	}

	if err := os.Chdir("/"); err != nil {
		return err
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return err
	}

	return nil
}
