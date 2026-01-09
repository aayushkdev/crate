package fs

import (
	"fmt"
	"os"
	"syscall"
)

func mountDev(rootless bool, hostFDs map[string]*os.File) error {
	if err := os.MkdirAll("/dev", 0755); err != nil {
		return err
	}

	if err := syscall.Mount(
		"tmpfs",
		"/dev",
		"tmpfs",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_STRICTATIME,
		"mode=755",
	); err != nil {
		return err
	}

	_ = syscall.Mount("", "/dev", "", syscall.MS_PRIVATE|syscall.MS_REC, "")

	if rootless {
		for name, f := range hostFDs {
			dst := "/dev/" + name

			file, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR, 0666)
			if err != nil {
				return err
			}
			file.Close()

			src := fmt.Sprintf("/proc/self/fd/%d", f.Fd())
			if err := syscall.Mount(src, dst, "", syscall.MS_BIND, ""); err != nil {
				return fmt.Errorf("bind %s: %w", name, err)
			}

			syscall.Mount(
				"",
				dst,
				"",
				syscall.MS_BIND|syscall.MS_REMOUNT|
					syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC,
				"",
			)
		}
	} else {
		mknodChar("/dev/null", 1, 3, 0666)
		mknodChar("/dev/zero", 1, 5, 0666)
		mknodChar("/dev/random", 1, 8, 0666)
		mknodChar("/dev/urandom", 1, 9, 0666)
		mknodChar("/dev/full", 1, 7, 0666)
		mknodChar("/dev/tty", 5, 0, 0666)
	}

	os.MkdirAll("/dev/shm", 1777)
	syscall.Mount(
		"shm",
		"/dev/shm",
		"tmpfs",
		syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_NOEXEC,
		"mode=1777,size=64m",
	)

	os.MkdirAll("/dev/pts", 0755)

	gid := os.Getgid()
	opts := fmt.Sprintf("newinstance,ptmxmode=0666,mode=620,gid=%d", gid)

	if err := syscall.Mount(
		"devpts",
		"/dev/pts",
		"devpts",
		syscall.MS_NOSUID|syscall.MS_NOEXEC,
		opts,
	); err != nil {
		return err
	}

	os.Remove("/dev/ptmx")
	return os.Symlink("pts/ptmx", "/dev/ptmx")
}

func mknodChar(path string, major, minor int, perm uint32) {
	dev := int((major << 8) | minor)
	mode := uint32(syscall.S_IFCHR | perm)

	syscall.Mknod(path, mode, dev)
}
