package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/aayushkdev/crate/internal/container"
	cratenet "github.com/aayushkdev/crate/internal/net"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

const (
	execConfigFD   = 3
	execRootPIDEnv = "CRATE_EXEC_ROOT_PID"
)

type execChildConfig struct {
	Env        []string `json:"env"`
	User       string   `json:"user,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
	TTY        bool     `json:"tty,omitempty"`
}

func Exec(containerID string, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}

	state, _, err := readRunningContainer(containerID)
	if err != nil {
		return err
	}

	cmd := selfCommand("exec-init", state.ID, command)
	if useExecPTY(command) {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		ptmx, err := startAttachedNoCTTY(cmd)
		if err != nil {
			return err
		}
		if err := relayAttached(ptmx, os.Stdout); err != nil {
			cleanupErr := killProcessTree(cmd.Process.Pid, syscall.SIGKILL)
			waitErr := cmd.Wait()
			return errors.Join(err, cleanupErr, waitErr)
		}

		return cmd.Wait()
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
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

	return runNsenterExec(state, cfg, command, useExecPTY(command))
}

func runNsenterExec(state *container.State, cfg *container.Config, command []string, tty bool) error {
	nsenterPath, err := osexec.LookPath("nsenter")
	if err != nil {
		return fmt.Errorf("exec requires nsenter: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve self executable: %w", err)
	}

	execCfg, err := createExecConfigFile(cfg, tty)
	if err != nil {
		return err
	}
	defer execCfg.Close()

	args := []string{
		"--target", strconv.Itoa(state.PID),
		"--mount",
		"--uts",
	}
	if cfg.Rootless {
		args = append(args, "--user")
	}
	if shouldJoinNetNS(state, cfg) {
		args = append(args, "--net")
	}
	args = append(args, "--pid")
	args = append(args, "--preserve-credentials", "--keep-caps", "--", exePath, "exec-child", state.ID)
	args = append(args, command...)

	cmd := osexec.Command(nsenterPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(
		os.Environ(),
		execRootPIDEnv+"="+strconv.Itoa(state.PID),
	)
	cmd.ExtraFiles = []*os.File{execCfg}

	return cmd.Run()
}

func runExecChild(_ string, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}

	cfg, err := readExecChildConfig()
	if err != nil {
		return err
	}

	if err := enterExecRootFromEnv(); err != nil {
		return err
	}
	if err := chdirOrRoot(cfg.WorkingDir); err != nil {
		return err
	}
	if cfg.TTY {
		if err := claimExecTTY(); err != nil {
			return err
		}
		if err := ensureExecDevTTY(); err != nil {
			return err
		}
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

func claimExecTTY() error {
	if !isTerminal(os.Stdin) {
		return fmt.Errorf("exec tty requested without terminal stdin")
	}
	if _, err := unix.Setsid(); err != nil {
		return fmt.Errorf("create exec tty session: %w", err)
	}
	if err := unix.IoctlSetInt(int(os.Stdin.Fd()), unix.TIOCSCTTY, 0); err != nil {
		return fmt.Errorf("set exec controlling tty: %w", err)
	}

	return nil
}

func ensureExecDevTTY() error {
	if _, err := os.Lstat("/dev/tty"); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Symlink("/proc/self/fd/0", "/dev/tty"); err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

func enterExecRootFromEnv() error {
	rawPID := os.Getenv(execRootPIDEnv)
	if rawPID == "" {
		return fmt.Errorf("%s is not set", execRootPIDEnv)
	}
	pid, err := strconv.Atoi(rawPID)
	if err != nil || pid <= 0 {
		return fmt.Errorf("invalid %s %q", execRootPIDEnv, rawPID)
	}

	path := fmt.Sprintf("/proc/%d/root", pid)
	root, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open container root: %w", err)
	}
	defer root.Close()

	return enterExecRoot(root)
}

func enterExecRoot(root *os.File) error {
	if err := unix.Fchdir(int(root.Fd())); err != nil {
		return fmt.Errorf("enter container root: %w", err)
	}
	if err := unix.Chroot("."); err != nil {
		return fmt.Errorf("chroot container root: %w", err)
	}
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir /: %w", err)
	}

	return nil
}

func createExecConfigFile(cfg *container.Config, tty bool) (*os.File, error) {
	f, err := os.CreateTemp("", "crate-exec-config-*")
	if err != nil {
		return nil, fmt.Errorf("create exec config: %w", err)
	}
	if err := os.Remove(f.Name()); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("unlink exec config: %w", err)
	}

	execCfg, err := marshalExecConfig(cfg, tty)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if _, err := f.Write(execCfg); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("write exec config: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("rewind exec config: %w", err)
	}

	return f, nil
}

func marshalExecConfig(cfg *container.Config, tty bool) ([]byte, error) {
	execCfg := execChildConfig{
		Env:        cfg.Env,
		User:       cfg.User,
		WorkingDir: cfg.WorkingDir,
		TTY:        tty,
	}
	data, err := json.Marshal(execCfg)
	if err != nil {
		return nil, fmt.Errorf("encode exec config: %w", err)
	}

	return data, nil
}

func readExecChildConfig() (*execChildConfig, error) {
	f := os.NewFile(uintptr(execConfigFD), "exec-config")
	if f == nil {
		return nil, fmt.Errorf("exec config fd is unavailable")
	}
	defer f.Close()
	var cfg execChildConfig
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("read exec config: %w", err)
	}

	return &cfg, nil
}

func useExecPTY(command []string) bool {
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return false
	}
	if len(command) != 1 {
		return false
	}

	switch filepath.Base(command[0]) {
	case "sh", "ash", "bash", "dash", "zsh", "fish":
		return true
	default:
		return false
	}
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}

	return term.IsTerminal(int(f.Fd()))
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
