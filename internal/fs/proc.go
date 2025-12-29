package fs

import (
	"os"
	"syscall"
)

func mountProc() error {
	if err := os.MkdirAll("/proc", 0555); err != nil {
		return err
	}

	return syscall.Mount("proc", "/proc", "proc", syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV, "")
}
