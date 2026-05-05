package net

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	storage "github.com/aayushkdev/crate/internal/storage"
)

func Setup(containerID string, pid int, cfg Config) error {
	cfg = NormalizeConfig(cfg, true)
	if err := ValidateConfig(cfg, true); err != nil {
		return err
	}

	if !RequiresHelper(cfg) {
		return nil
	}

	if err := prepareRootfsFiles(storage.ContainerRootfsPath(containerID)); err != nil {
		return err
	}

	logFile, err := openLogFile(containerID)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(
		"pasta",
		pastaArgs(pid, cfg)...,
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return err
	}

	if err := writeState(containerID, &State{
		Mode:          string(cfg.Mode),
		Backend:       "pasta",
		HelperPID:     cmd.Process.Pid,
		InterfaceName: cfg.InterfaceName,
		LogPath:       storage.NetworkLogPath(containerID),
		Publish:       append([]PublishedPort(nil), cfg.Publish...),
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

func pastaArgs(pid int, cfg Config) []string {
	args := []string{
		"--config-net",
		"--ns-ifname", cfg.InterfaceName,
	}

	if len(cfg.Publish) > 0 {
		tcpSpec := PastaPortSpec(cfg.Publish, "tcp")
		if tcpSpec == "" {
			tcpSpec = "none"
		}
		args = append(args, "-t", tcpSpec)

		udpSpec := PastaPortSpec(cfg.Publish, "udp")
		if udpSpec == "" {
			udpSpec = "none"
		}
		args = append(args, "-u", udpSpec)
	}

	args = append(args, strconv.Itoa(pid))
	return args
}

func Teardown(containerID string) error {
	state, err := readState(containerID)
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

	return removeState(containerID)
}

func openLogFile(containerID string) (*os.File, error) {
	path := storage.NetworkLogPath(containerID)
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
		return err
	case <-time.After(timeout):
		return nil
	}
}
