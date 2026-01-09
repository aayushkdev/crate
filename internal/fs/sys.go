package fs

import "syscall"
import "os"

func mountSys(rootless bool) error {
	if rootless {
		return os.MkdirAll("/sys", 0555)
	}
	if err := os.MkdirAll("/sys", 0555); err != nil {
		return err
	}

	return syscall.Mount(
		"sysfs",
		"/sys",
		"sysfs",
		syscall.MS_RDONLY|
			syscall.MS_NOSUID|
			syscall.MS_NODEV|
			syscall.MS_NOEXEC,
		"",
	)
}
