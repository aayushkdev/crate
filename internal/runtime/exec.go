package runtime

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	goruntime "runtime"
	"strconv"
	"syscall"

	"github.com/aayushkdev/crate/internal/container"
	cratenet "github.com/aayushkdev/crate/internal/net"
	"golang.org/x/sys/unix"
)

const (
	execRootFD     = 3
	execConfigFD   = 4
	execConfigEnv  = "CRATE_EXEC_CONFIG"
	execRootPIDEnv = "CRATE_EXEC_ROOT_PID"
)

type execChildConfig struct {
	Env        []string `json:"env"`
	User       string   `json:"user,omitempty"`
	WorkingDir string   `json:"working_dir,omitempty"`
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

	if cfg.Rootless {
		return runRootlessExecInit(state, cfg, command)
	}

	root, err := os.Open(fmt.Sprintf("/proc/%d/root", state.PID))
	if err != nil {
		return fmt.Errorf("open container root: %w", err)
	}
	defer root.Close()

	execCfg, err := createExecConfigFile(cfg)
	if err != nil {
		return err
	}
	defer execCfg.Close()

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
	cmd.ExtraFiles = []*os.File{root, execCfg}

	return cmd.Run()
}

func runRootlessExecInit(state *container.State, cfg *container.Config, command []string) error {
	nsenterPath, err := osexec.LookPath("nsenter")
	if err != nil {
		return fmt.Errorf("rootless exec requires nsenter: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve self executable: %w", err)
	}

	execCfg, err := marshalExecConfig(cfg)
	if err != nil {
		return err
	}

	args := []string{
		"--target", strconv.Itoa(state.PID),
		"--user",
		"--mount",
		"--uts",
	}
	if shouldJoinNetNS(state, cfg) {
		args = append(args, "--net")
	}
	args = append(args,
		"--pid",
		"--preserve-credentials",
		"--keep-caps",
		"--",
		exePath,
		"exec-child",
		state.ID,
	)
	args = append(args, command...)

	cmd := osexec.Command(nsenterPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(
		os.Environ(),
		execConfigEnv+"="+base64.StdEncoding.EncodeToString(execCfg),
		execRootPIDEnv+"="+strconv.Itoa(state.PID),
	)

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

	if rootPID := os.Getenv(execRootPIDEnv); rootPID != "" {
		if err := enterExecRootPath(fmt.Sprintf("/proc/%s/root", rootPID)); err != nil {
			return err
		}
	} else {
		if err := enterExecRootFD(); err != nil {
			return err
		}
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

func enterExecRootFD() error {
	root := os.NewFile(uintptr(execRootFD), "container-root")
	if root == nil {
		return fmt.Errorf("container root fd is unavailable")
	}
	defer root.Close()

	return enterExecRoot(root)
}

func enterExecRootPath(path string) error {
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

func createExecConfigFile(cfg *container.Config) (*os.File, error) {
	f, err := os.CreateTemp("", "crate-exec-config-*")
	if err != nil {
		return nil, fmt.Errorf("create exec config: %w", err)
	}
	if err := os.Remove(f.Name()); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("unlink exec config: %w", err)
	}

	execCfg, err := marshalExecConfig(cfg)
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

func marshalExecConfig(cfg *container.Config) ([]byte, error) {
	execCfg := execChildConfig{
		Env:        cfg.Env,
		User:       cfg.User,
		WorkingDir: cfg.WorkingDir,
	}
	data, err := json.Marshal(execCfg)
	if err != nil {
		return nil, fmt.Errorf("encode exec config: %w", err)
	}

	return data, nil
}

func readExecChildConfig() (*execChildConfig, error) {
	if encoded := os.Getenv(execConfigEnv); encoded != "" {
		data, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode exec config: %w", err)
		}

		var cfg execChildConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("read exec config: %w", err)
		}

		return &cfg, nil
	}

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
