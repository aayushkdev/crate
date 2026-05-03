package net

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	filestate "github.com/aayushkdev/crate/internal/state"
)

func Setup(containerID string, pid int, cfg Config) error {
	cfg = NormalizeConfig(cfg, true)
	if err := ValidateConfig(cfg, true); err != nil {
		return err
	}

	if !RequiresHelper(cfg) {
		return nil
	}

	if err := prepareRootfsFiles(filestate.ContainerRootfsPath(containerID)); err != nil {
		return err
	}

	logFile, err := openLogFile(containerID)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(
		"pasta",
		"--config-net",
		"--ns-ifname", cfg.InterfaceName,
		strconv.Itoa(pid),
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := filestate.WriteNetwork(containerID, &filestate.Network{
		Mode:          string(cfg.Mode),
		Backend:       "pasta",
		HelperPID:     cmd.Process.Pid,
		InterfaceName: cfg.InterfaceName,
		LogPath:       filestate.NetworkLogPath(containerID),
	}); err != nil {
		_ = cmd.Process.Kill()
		return err
	}

	if err := waitForHelper(cmd, 200*time.Millisecond); err != nil {
		_ = removeState(containerID)
		return fmt.Errorf("start pasta: %w", err)
	}

	return nil
}

func Teardown(containerID string) error {
	state, err := filestate.ReadNetwork(containerID)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if state.HelperPID > 0 {
		err = syscall.Kill(state.HelperPID, syscall.SIGTERM)
		if err != nil && err != syscall.ESRCH {
			return err
		}
	}

	return filestate.RemoveNetwork(containerID)
}

func openLogFile(containerID string) (*os.File, error) {
	path := filestate.NetworkLogPath(containerID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}

	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

func waitForHelper(cmd *exec.Cmd, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err == nil {
			return fmt.Errorf("helper exited unexpectedly")
		}
		return err
	case <-time.After(timeout):
		return nil
	}
}

func removeState(containerID string) error {
	return filestate.RemoveNetwork(containerID)
}
