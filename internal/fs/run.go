package fs

import "syscall"
import "os"

func mountRun() error {
	if err := os.MkdirAll("/run", 0755); err != nil {
		return err
	}

	return syscall.Mount(
		"tmpfs",
		"/run",
		"tmpfs",
		syscall.MS_NOSUID|syscall.MS_NODEV,
		"mode=755",
	)
}
