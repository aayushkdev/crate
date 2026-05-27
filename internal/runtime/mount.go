package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/aayushkdev/crate/internal/container"
)

func applyMounts(mounts []container.OpenedMount) error {
	for _, mount := range mounts {
		if err := prepareMountDestination(mount.Mount.Destination, mount.Info); err != nil {
			return err
		}

		source := fmt.Sprintf("/proc/self/fd/%d", mount.File.Fd())
		if err := syscall.Mount(source, mount.Mount.Destination, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
			return fmt.Errorf("bind volume %s to %s: %w", mount.Mount.Source, mount.Mount.Destination, err)
		}

		if mount.Mount.ReadOnly {
			flags := uintptr(syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_RDONLY | syscall.MS_REC)
			if err := syscall.Mount(source, mount.Mount.Destination, "", flags, ""); err != nil {
				return fmt.Errorf("remount volume %s read-only: %w", mount.Mount.Destination, err)
			}
		}
	}

	return nil
}

func prepareMountDestination(path string, sourceInfo os.FileInfo) error {
	info, err := os.Lstat(path)
	if err == nil {
		if sourceInfo.IsDir() && !info.IsDir() {
			return fmt.Errorf("volume destination %s exists and is not a directory", path)
		}
		if !sourceInfo.IsDir() && info.IsDir() {
			return fmt.Errorf("volume destination %s exists and is a directory", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat volume destination %s: %w", path, err)
	}

	if sourceInfo.IsDir() {
		if err := os.MkdirAll(path, sourceInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("create volume destination %s: %w", path, err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create volume destination parent %s: %w", filepath.Dir(path), err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL, sourceInfo.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create volume destination %s: %w", path, err)
	}
	return file.Close()
}
