package runtime

import (
	"syscall"
	"time"

	"github.com/aayushkdev/crate/internal/container"
)

func killProcessGroup(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return syscall.EINVAL
	}

	return syscall.Kill(-pid, sig)
}

func waitForExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !container.ProcessAlive(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	return !container.ProcessAlive(pid)
}
