package runtime

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aayushkdev/crate/internal/container"
)

func launchContainer(containerID string, command []string, cfg *container.Config, attach bool, wait bool) error {
	argv, err := container.ResolveEntrypoint(cfg, command)
	if err != nil {
		return err
	}

	logPath := container.LogPath(containerID)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	args := append([]string{"init", containerID}, command...)
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	if cfg.Rootless {
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS

		cmd.SysProcAttr.UidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		}
		cmd.SysProcAttr.GidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		}
		cmd.SysProcAttr.GidMappingsEnableSetgroups = false
	} else {
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS
	}

	if attach {
		cmd.Stdin = os.Stdin
		cmd.Stdout = io.MultiWriter(os.Stdout, logFile)
		cmd.Stderr = io.MultiWriter(os.Stderr, logFile)
	} else {
		cmd.SysProcAttr.Setpgid = true
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	state := &container.State{
		ID:        containerID,
		Image:     cfg.Image,
		Command:   argv,
		Status:    container.StatusRunning,
		PID:       cmd.Process.Pid,
		LogPath:   logPath,
		StartedAt: time.Now().UTC(),
	}
	if err := container.UpdateState(containerID, func(s *container.State) {
		*s = *state
	}); err != nil {
		_ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Process.Kill()
		return err
	}

	if !wait {
		return nil
	}

	waitErr := cmd.Wait()
	exitCode := exitCode(waitErr)
	status := container.StatusExited
	current, err := container.ReadState(containerID)
	if err == nil && current.Status == container.StatusStopping {
		status = container.StatusStopped
	}
	if err := container.UpdateState(containerID, func(s *container.State) {
		s.Status = status
		s.ExitCode = exitCode
		s.FinishedAt = time.Now().UTC()
	}); err != nil {
		return err
	}

	return waitErr
}

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

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return 1
}
