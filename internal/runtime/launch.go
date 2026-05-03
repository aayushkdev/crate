package runtime

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aayushkdev/crate/internal/container"
	cratenet "github.com/aayushkdev/crate/internal/net"
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

	var syncW *os.File
	if cratenet.RequiresNetNS(cfg.Network) {
		syncR, pipeW, err := os.Pipe()
		if err != nil {
			return err
		}
		defer syncR.Close()

		cmd.ExtraFiles = append(cmd.ExtraFiles, syncR)
		cmd.Env = append(os.Environ(), cratenet.SyncEnv(3))
		syncW = pipeW
	}
	if cfg.Rootless {
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS
		if cratenet.RequiresNetNS(cfg.Network) {
			cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWNET
		}

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
		if cratenet.RequiresNetNS(cfg.Network) {
			cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWNET
		}
	}

	if !attach {
		cmd.SysProcAttr.Setpgid = true
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	var ptmx *os.File
	if attach {
		ptmx, err = startAttached(cmd)
		if err != nil {
			return err
		}
	} else {
		if err := cmd.Start(); err != nil {
			return err
		}
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

	if syncW != nil {
		if err := cratenet.Setup(containerID, cmd.Process.Pid, cfg.Network); err != nil {
			_ = syncW.Close()
			_ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Process.Kill()
			return err
		}

		if _, err := syncW.Write([]byte{1}); err != nil {
			_ = syncW.Close()
			_ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Process.Kill()
			_ = cratenet.Teardown(containerID)
			return err
		}

		if err := syncW.Close(); err != nil {
			_ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Process.Kill()
			_ = cratenet.Teardown(containerID)
			return err
		}
	}

	if !wait {
		return nil
	}

	if attach {
		if err := relayAttached(ptmx, logFile); err != nil {
			_ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Process.Kill()
			return err
		}
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

	if err := cratenet.Teardown(containerID); err != nil {
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
