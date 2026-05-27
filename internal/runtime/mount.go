package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/aayushkdev/crate/internal/container"
)

type openedMount struct {
	mount container.Mount
	file  *os.File
	info  os.FileInfo
}

func openMounts(mounts []container.Mount) ([]openedMount, error) {
	opened := make([]openedMount, 0, len(mounts))
	for _, mount := range mounts {
		file, err := os.Open(mount.Source)
		if err != nil {
			closeMounts(opened)
			return nil, fmt.Errorf("open volume source %s: %w", mount.Source, err)
		}

		info, err := file.Stat()
		if err != nil {
			file.Close()
			closeMounts(opened)
			return nil, fmt.Errorf("stat volume source %s: %w", mount.Source, err)
		}

		opened = append(opened, openedMount{
			mount: mount,
			file:  file,
			info:  info,
		})
	}

	return opened, nil
}

func closeMounts(mounts []openedMount) {
	for _, mount := range mounts {
		_ = mount.file.Close()
	}
}

func applyMounts(mounts []openedMount) error {
	for _, mount := range mounts {
		if err := prepareMountDestination(mount.mount.Destination, mount.info); err != nil {
			return err
		}

		source := fmt.Sprintf("/proc/self/fd/%d", mount.file.Fd())
		if err := syscall.Mount(source, mount.mount.Destination, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
			return fmt.Errorf("bind volume %s to %s: %w", mount.mount.Source, mount.mount.Destination, err)
		}

		if mount.mount.ReadOnly {
			flags := uintptr(syscall.MS_BIND | syscall.MS_REMOUNT | syscall.MS_RDONLY | syscall.MS_REC)
			if err := syscall.Mount(source, mount.mount.Destination, "", flags, ""); err != nil {
				return fmt.Errorf("remount volume %s read-only: %w", mount.mount.Destination, err)
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
