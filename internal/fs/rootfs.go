package fs

import (
	"os"
	"path/filepath"
	"syscall"
)

func setupRootfs(rootfs string) error {
	if err := syscall.Mount(rootfs, rootfs, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}

	putold := filepath.Join(rootfs, ".oldroot")
	if err := os.MkdirAll(putold, 0700); err != nil {
		return err
	}

	if err := syscall.PivotRoot(rootfs, putold); err != nil {
		return err
	}

	if err := syscall.Chdir("/"); err != nil {
		return err
	}

	if err := syscall.Unmount("/.oldroot", syscall.MNT_DETACH); err != nil {
		return err
	}

	if err := os.RemoveAll("/.oldroot"); err != nil {
		return err
	}

	return nil
}
