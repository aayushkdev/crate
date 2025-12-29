package fs

import (
	"syscall"
)

func Setup(rootfs string) error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return err
	}

	if err := setupRootfs(rootfs); err != nil {
		return err
	}

	if err := mountProc(); err != nil {
		return err
	}

	return nil
}
