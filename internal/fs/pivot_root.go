package fs

import (
	"os"
	"path/filepath"
	"syscall"
)

func PivotRoot(newRoot string) error {
	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}

	putOld := filepath.Join(newRoot, ".pivot_root")
	if err := os.MkdirAll(putOld, 0700); err != nil {
		return err
	}

	if err := syscall.PivotRoot(newRoot, putOld); err != nil {
		return err
	}

	if err := syscall.Chdir("/"); err != nil {
		return err
	}

	putOld = "/.pivot_root"
	if err := syscall.Unmount(putOld, syscall.MNT_DETACH); err != nil {
		return err
	}

	if err := os.RemoveAll(putOld); err != nil {
		return err
	}

	return nil
}
