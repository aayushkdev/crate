package fs

import (
	"os"
	"path/filepath"
	"syscall"
)

func setupRootfs(rootfs string, rootless bool) error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return err
	}
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}
	if rootless {
		if err := setupChroot(rootfs); err != nil {
			return err
		}
	} else {
		if err := setupPivotRoot(rootfs); err != nil {
			return err
		}
	}

	return os.Chdir("/")
}

func setupPivotRoot(rootfs string) error {
	putold := filepath.Join(rootfs, ".oldroot")
	if err := os.MkdirAll(putold, 0700); err != nil {
		return err
	}

	if err := syscall.PivotRoot(rootfs, putold); err != nil {
		return err
	}

	if err := syscall.Unmount("/.oldroot", syscall.MNT_DETACH); err != nil {
		return err
	}

	return os.RemoveAll("/.oldroot")
}

func setupChroot(rootfs string) error {
	return syscall.Chroot(rootfs)
}
