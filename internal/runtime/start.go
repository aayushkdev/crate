package runtime

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/aayushkdev/crate/internal/container"
)

func Start(containerID string, command []string) error {
	cfg, err := container.ReadConfig(containerID)
	if err != nil {
		return err
	}

	args := append([]string{"init", containerID}, command...)
	cmd := exec.Command("/proc/self/exe", args...)

	sys := &syscall.SysProcAttr{}
	if cfg.Rootless {
		sys.Cloneflags = syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS

		sys.UidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getuid(), Size: 1},
		}
		sys.GidMappings = []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: os.Getgid(), Size: 1},
		}

		sys.GidMappingsEnableSetgroups = false
	} else {
		sys.Cloneflags = syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS
	}

	cmd.SysProcAttr = sys
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
