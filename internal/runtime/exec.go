package runtime

import (
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	goruntime "runtime"
	"syscall"

	"github.com/aayushkdev/crate/internal/container"
	cratenet "github.com/aayushkdev/crate/internal/net"
	"golang.org/x/sys/unix"
)

const execRootFD = 3

func Exec(containerID string, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}

	state, _, err := readRunningContainer(containerID)
	if err != nil {
		return err
	}

	cmd := selfCommand("exec-init", state.ID, command)
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	ptmx, err := startAttached(cmd)
	if err != nil {
		return err
	}
	if err := relayAttached(ptmx, os.Stdout); err != nil {
		_ = killProcessGroup(cmd.Process.Pid, syscall.SIGKILL)
		_ = cmd.Process.Kill()
		waitErr := cmd.Wait()
		return errors.Join(err, waitErr)
	}

	return cmd.Wait()
}

func ExecInit(containerID string, command []string) {
	fatalExec(runExecInit(containerID, command))
}

func ExecChild(containerID string, command []string) {
	fatalExec(runExecChild(containerID, command))
}

func runExecInit(containerID string, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}

	state, cfg, err := readRunningContainer(containerID)
	if err != nil {
		return err
	}

	root, err := os.Open(fmt.Sprintf("/proc/%d/root", state.PID))
	if err != nil {
		return fmt.Errorf("open container root: %w", err)
	}
	defer root.Close()

	ns, err := openContainerNamespaces(state.PID, cfg.Rootless, shouldJoinNetNS(state, cfg))
	if err != nil {
		return err
	}
	defer ns.Close()

	goruntime.LockOSThread()
	defer goruntime.UnlockOSThread()

	if err := joinContainerNamespaces(ns); err != nil {
		return err
	}

	cmd := selfCommand("exec-child", state.ID, command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{root}

	return cmd.Run()
}

func runExecChild(containerID string, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}

	cfg, err := container.ReadConfig(containerID)
	if err != nil {
		return err
	}

	root := os.NewFile(uintptr(execRootFD), "container-root")
	if root == nil {
		return fmt.Errorf("container root fd is unavailable")
	}
	defer root.Close()

	if err := unix.Fchdir(int(root.Fd())); err != nil {
		return fmt.Errorf("enter container root: %w", err)
	}
	if err := unix.Chroot("."); err != nil {
		return fmt.Errorf("chroot container root: %w", err)
	}
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir /: %w", err)
	}
	if err := chdirOrRoot(cfg.WorkingDir); err != nil {
		return err
	}
	if err := container.ApplyUser(cfg.User); err != nil {
		return err
	}

	execPath, err := container.ResolvePath(command[0], cfg.Env)
	if err != nil {
		return err
	}

	return syscall.Exec(execPath, command, cfg.Env)
}

func readRunningContainer(containerID string) (*container.State, *container.Config, error) {
	state, err := container.RefreshState(containerID)
	if err != nil {
		return nil, nil, err
	}
	if state.Status != container.StatusRunning || !container.ProcessAlive(state.PID) {
		return nil, nil, fmt.Errorf("container %s is not running", containerID)
	}

	cfg, err := container.ReadConfig(state.ID)
	if err != nil {
		return nil, nil, err
	}

	return state, cfg, nil
}

func shouldJoinNetNS(state *container.State, cfg *container.Config) bool {
	mode := cratenet.Mode(state.NetworkMode)
	if mode == "" {
		mode = cfg.Network.Mode
	}

	return cratenet.RequiresNetNS(cratenet.Config{Mode: mode})
}

func fatalExec(err error) {
	if err == nil {
		return
	}
	if exitErr, ok := err.(*osexec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}

	fmt.Fprintln(os.Stderr, "crate: exec failed", err)
	os.Exit(1)
}
