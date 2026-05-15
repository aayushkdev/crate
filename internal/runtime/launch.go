package runtime

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	if cfg.ID != "" {
		containerID = cfg.ID
	}

	netCfg, warning, err := cratenet.ResolveRuntimeConfig(cfg.Network, cfg.Rootless)
	if err != nil {
		return err
	}
	if warning != "" {
		fmt.Fprintf(os.Stderr, "crate: warning: %s\n", warning)
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
	cmd.Env = append(os.Environ(), cratenet.ModeEnv(netCfg.Mode))

	var syncW *os.File
	if cfg.Rootless || cratenet.RequiresNetNS(netCfg) {
		syncR, pipeW, err := os.Pipe()
		if err != nil {
			return err
		}
		defer syncR.Close()

		cmd.ExtraFiles = append(cmd.ExtraFiles, syncR)
		syncFD := 3 + len(cmd.ExtraFiles) - 1
		cmd.Env = append(cmd.Env, cratenet.SyncEnv(syncFD))
		syncW = pipeW
	}
	if cfg.Rootless {
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS
		if cratenet.RequiresNetNS(netCfg) {
			cmd.SysProcAttr.Cloneflags |= syscall.CLONE_NEWNET
		}
	} else {
		cmd.SysProcAttr.Cloneflags = syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS
		if cratenet.RequiresNetNS(netCfg) {
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
		ID:          containerID,
		Name:        cfg.Name,
		Image:       cfg.Image,
		Command:     argv,
		Status:      container.StatusRunning,
		PID:         cmd.Process.Pid,
		LogPath:     logPath,
		NetworkMode: string(netCfg.Mode),
		StartedAt:   time.Now().UTC(),
	}
	if cfg.Rootless {
		if err := configureRootlessUserNS(cmd.Process.Pid); err != nil {
			cleanupLaunch(syncW, cmd.Process.Pid, containerID)
			return err
		}
	}

	if err := container.UpdateState(containerID, func(s *container.State) {
		*s = *state
	}); err != nil {
		cleanupLaunch(syncW, cmd.Process.Pid, containerID)
		return err
	}

	if syncW != nil {
		if err := cratenet.Setup(containerID, cmd.Process.Pid, netCfg); err != nil {
			cleanupLaunch(syncW, cmd.Process.Pid, containerID)
			return err
		}

		if _, err := syncW.Write([]byte{1}); err != nil {
			cleanupLaunch(syncW, cmd.Process.Pid, containerID)
			return err
		}

		if err := syncW.Close(); err != nil {
			cleanupLaunch(nil, cmd.Process.Pid, containerID)
			return err
		}
	}

	if !wait {
		go reapDetached(containerID, cmd)
		return nil
	}

	if attach {
		if err := relayAttached(ptmx, logFile); err != nil {
			_ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
			_ = cmd.Process.Kill()
			waitErr := cmd.Wait()
			if finalizeErr := finalizeContainer(containerID, waitErr); finalizeErr != nil {
				log.Printf("crate: finalize attached container %s after relay error: %v", containerID, finalizeErr)
			}
			return err
		}
	}

	return finalizeContainer(containerID, cmd.Wait())
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

func cleanupLaunch(syncW *os.File, pid int, containerID string) {
	if syncW != nil {
		_ = syncW.Close()
	}
	if pid > 0 {
		_ = killProcessGroup(pid, syscall.SIGKILL)
	}
	_ = cratenet.Teardown(containerID)
}

func reapDetached(containerID string, cmd *exec.Cmd) {
	if err := finalizeContainer(containerID, cmd.Wait()); err != nil {
		log.Printf("crate: finalize detached container %s: %v", containerID, err)
	}
}

func finalizeContainer(containerID string, waitErr error) error {
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
	warnPrivilegeDropFailure(containerID, exitCode)

	return waitErr
}

func warnPrivilegeDropFailure(containerID string, exitCode int) {
	if exitCode != 127 {
		return
	}

	cfg, err := container.ReadConfig(containerID)
	if err != nil {
		return
	}
	if !cfg.Rootless || !container.IsRootUserSpec(cfg.User) {
		return
	}

	data, err := os.ReadFile(container.LogPath(containerID))
	if err != nil {
		return
	}
	if !strings.Contains(string(data), "setgroups failed") {
		return
	}

	fmt.Fprintf(
		os.Stderr,
		"crate: warning: image %q failed to drop privileges in rootless mode (setgroups disabled). Try --user if the image can start as its final service user\n",
		cfg.Image,
	)
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
