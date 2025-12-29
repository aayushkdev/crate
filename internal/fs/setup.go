package fs

import (
	"os"
	"syscall"
)

func Setup(rootfs string) error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return err
	}

	if err := PivotRoot(rootfs); err != nil {
		return err
	}

	if err := os.MkdirAll("/proc", 0555); err != nil {
		return err
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return err
	}

	return nil
}
