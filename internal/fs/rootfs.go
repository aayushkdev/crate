package fs

import (
	"syscall"
)

func setupRootfs(rootfs string) error {
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}

	if err := syscall.Chroot(rootfs); err != nil {
		return err
	}

	if err := syscall.Chdir("/"); err != nil {
		return err
	}

	return nil
}
