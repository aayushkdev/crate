package net

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

const ifnamsiz = 16

type ifreqFlags struct {
	Name  [ifnamsiz]byte
	Flags int16
	_     [24 - ifnamsiz - 2]byte
}

func BringUpLoopback() error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	var req ifreqFlags
	copy(req.Name[:], "lo")

	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCGIFFLAGS),
		uintptr(unsafe.Pointer(&req)),
	); errno != 0 {
		return fmt.Errorf("read loopback flags: %w", errno)
	}

	req.Flags |= unix.IFF_UP

	if _, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCSIFFLAGS),
		uintptr(unsafe.Pointer(&req)),
	); errno != 0 {
		return fmt.Errorf("bring up loopback: %w", errno)
	}

	return nil
}
