package runtime

import (
	"fmt"
	"io"
	"log"
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

	cmd := selfCommand("init", containerID, command)
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
			cleanupLaunch(syncW, cmd, containerID)
			return err
		}
	}

	if err := container.UpdateState(containerID, func(s *container.State) {
		createdAt := s.CreatedAt
		*s = *state
		s.CreatedAt = createdAt
	}); err != nil {
		cleanupLaunch(syncW, cmd, containerID)
		return err
	}

	if syncW != nil {
		if err := cratenet.Setup(containerID, cmd.Process.Pid, netCfg); err != nil {
			cleanupLaunch(syncW, cmd, containerID)
			return err
		}

		if _, err := syncW.Write([]byte{1}); err != nil {
			cleanupLaunch(syncW, cmd, containerID)
			return err
		}

		if err := syncW.Close(); err != nil {
			cleanupLaunch(nil, cmd, containerID)
			return err
		}
	}

	if !wait {
		go reapDetached(containerID, cmd)
		return nil
	}

	if attach {
		if err := relayAttached(ptmx, io.MultiWriter(os.Stdout, logFile)); err != nil {
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

func cleanupLaunch(syncW *os.File, cmd *exec.Cmd, containerID string) {
	if syncW != nil {
		_ = syncW.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
	_ = cratenet.Teardown(containerID)
}
