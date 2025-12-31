package runtime

import (
	"os"
	"os/exec"
	"syscall"
)

func LaunchContainer(image string, command []string) error {
	isRoot := os.Geteuid() == 0
	root := "0"
	if isRoot {
		root = "1"
	}
	args := append([]string{"init", root, image}, command...)
	cmd := exec.Command("/proc/self/exe", args...)

	sys := &syscall.SysProcAttr{}

	if isRoot {
		sys.Cloneflags =
			syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS
	} else {
		sys.Cloneflags =
			syscall.CLONE_NEWUSER |
				syscall.CLONE_NEWUTS |
				syscall.CLONE_NEWPID |
				syscall.CLONE_NEWNS

		sys.UidMappings = []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		}

		sys.GidMappings = []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		}
	}

	cmd.SysProcAttr = sys
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
