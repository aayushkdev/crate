package net

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

const syncFDEnv = "CRATE_SYNC_FD"

func SyncEnv(fd int) string {
	return syncFDEnv + "=" + strconv.Itoa(fd)
}

func WaitForParent() error {
	value := os.Getenv(syncFDEnv)
	if value == "" {
		return nil
	}

	fd, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("parse %s: %w", syncFDEnv, err)
	}

	file := os.NewFile(uintptr(fd), "crate-sync")
	if file == nil {
		return fmt.Errorf("open sync fd %d", fd)
	}
	defer file.Close()

	buf := make([]byte, 1)
	n, err := file.Read(buf)
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("network setup aborted")
		}
		return err
	}
	if n != 1 || buf[0] != 1 {
		return fmt.Errorf("network setup handshake failed")
	}

	return nil
}

func WaitForInterface(name string, timeout time.Duration) error {
	if name == "" {
		return nil
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if interfaceExists(name) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("network interface %q did not appear", name)
}
