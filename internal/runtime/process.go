package runtime

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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

func killProcessTree(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return syscall.EINVAL
	}

	descendants, err := processDescendants(pid)
	if err != nil {
		return errors.Join(err, killProcessGroup(pid, sig))
	}

	pids := append([]int{pid}, descendants...)
	pgids := make(map[int]struct{}, len(pids))
	var killErr error
	for _, p := range pids {
		pgid, err := syscall.Getpgid(p)
		if err != nil {
			if !errors.Is(err, syscall.ESRCH) {
				killErr = errors.Join(killErr, fmt.Errorf("get process group %d: %w", p, err))
			}
			continue
		}
		if pgid > 0 {
			pgids[pgid] = struct{}{}
		}
	}

	for pgid := range pgids {
		if err := syscall.Kill(-pgid, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
			killErr = errors.Join(killErr, fmt.Errorf("kill process group %d: %w", pgid, err))
		}
	}
	for _, p := range pids {
		if err := syscall.Kill(p, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
			killErr = errors.Join(killErr, fmt.Errorf("kill process %d: %w", p, err))
		}
	}

	return killErr
}

func processDescendants(root int) ([]int, error) {
	parents, err := processParents()
	if err != nil {
		return nil, err
	}

	children := make(map[int][]int)
	for pid, ppid := range parents {
		children[ppid] = append(children[ppid], pid)
	}

	var descendants []int
	queue := append([]int(nil), children[root]...)
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		descendants = append(descendants, pid)
		queue = append(queue, children[pid]...)
	}

	return descendants, nil
}

func processParents() (map[int]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	parents := make(map[int]int)
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		ppid, err := processParent(pid)
		if err != nil {
			continue
		}
		parents[pid] = ppid
	}

	return parents, nil
}

func processParent(pid int) (int, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "PPid:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("parse process parent for %d", pid)
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			return 0, fmt.Errorf("parse process parent for %d: %w", pid, err)
		}
		return ppid, nil
	}

	return 0, fmt.Errorf("process parent missing for %d", pid)
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
