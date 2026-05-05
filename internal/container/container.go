package container

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/aayushkdev/crate/internal/fs"
	cratenet "github.com/aayushkdev/crate/internal/net"
	storage "github.com/aayushkdev/crate/internal/storage"
)

const initMappedEnv = "CRATE_INIT_MAPPED"

func InitContainer(containerID string, command []string) {
	cfg, err := ReadConfig(containerID)
	Fatal(err)

	if os.Getenv(initMappedEnv) != "1" {
		Fatal(cratenet.WaitForParent())
		if cfg.Rootless {
			Fatal(reexecMappedInit(containerID, command))
		}
	}

	cfg.Network = cratenet.ApplyModeOverride(cfg.Network)
	rootfs := storage.ContainerRootfsPath(containerID)

	Fatal(syscall.Sethostname([]byte("crate")))

	Fatal(fs.Setup(rootfs, cfg.Rootless))
	if cfg.Network.Mode == cratenet.ModeNone || cfg.Network.Mode == cratenet.ModePrivate {
		Fatal(cratenet.BringUpLoopback())
	}
	if cfg.Network.Mode == cratenet.ModePrivate {
		Fatal(cratenet.WaitForInterface(cfg.Network.InterfaceName, 5*time.Second))
	}
	Fatal(ApplyUser(cfg.User))

	if len(command) == 0 {
		command = cfg.Cmd
	}
	env := cfg.Env

	cmd, err := ResolveEntrypoint(cfg, command)
	Fatal(err)

	execPath, err := resolvePath(cmd[0], env)
	Fatal(err)

	Fatal(syscall.Exec(execPath, cmd, env))
}

func reexecMappedInit(containerID string, command []string) error {
	argv := append([]string{"/proc/self/exe", "init", containerID}, command...)

	env := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "CRATE_SYNC_FD=") {
			continue
		}
		if strings.HasPrefix(entry, initMappedEnv+"=") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, initMappedEnv+"=1")

	return syscall.Exec("/proc/self/exe", argv, env)
}

func Fatal(err error) {
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "crate: init exec failed", err)
		os.Exit(1)
	}
}
