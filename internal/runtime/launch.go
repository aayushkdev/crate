package runtime

import (
	"os"
	"os/exec"
	"syscall"
)

func LaunchContainer(args []string) error {
	uid := os.Getuid()
	gid := os.Getgid()

	cmd := exec.Command(
		"/proc/self/exe",
		append([]string{"init"}, args...)...,
	)

	//TODO: Support both rootless and rootful containers

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS,

		UidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: uid, Size: 1},
		},
		GidMappings: []syscall.SysProcIDMap{
			{ContainerID: 0, HostID: gid, Size: 1},
		},
		GidMappingsEnableSetgroups: false,
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
