package runtime

import (
	"os"
	"os/exec"
	"syscall"
)

func LaunchContainer(args []string) error {
	cmd := exec.Command(
		"/proc/self/exe",
		append([]string{"init"}, args...)...,
	)

	//TODO: use rootless container (CLONE_NEWUSER)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
