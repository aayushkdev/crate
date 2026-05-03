package storage

import (
	"os"
	"os/user"
	"path/filepath"
)

const (
	containersDirName   = "containers"
	rootfsDirName       = "rootfs"
	logsDirName         = "logs"
	containerConfigName = "config.json"
	containerStateName  = "state.json"
	containerLogName    = "container.log"
	networkStateName    = "network.json"
	networkLogName      = "network.log"
)

func ContainerDir(id string) string {
	return filepath.Join(ContainersDir(), id)
}

func ContainersDir() string {
	return filepath.Join(CrateRoot(), containersDirName)
}

func ContainerRootfsPath(id string) string {
	return filepath.Join(ContainerDir(id), rootfsDirName)
}

func ContainerConfigPath(id string) string {
	return filepath.Join(ContainerDir(id), containerConfigName)
}

func ContainerStatePath(id string) string {
	return filepath.Join(ContainerDir(id), containerStateName)
}

func ContainerLogPath(id string) string {
	return filepath.Join(ContainerDir(id), logsDirName, containerLogName)
}

func NetworkStatePath(id string) string {
	return filepath.Join(ContainerDir(id), networkStateName)
}

func NetworkLogPath(id string) string {
	return filepath.Join(ContainerDir(id), logsDirName, networkLogName)
}

func CrateRoot() string {
	var home string
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			home = u.HomeDir
		}
	}
	if home == "" {
		h, _ := os.UserHomeDir()
		home = h
	}

	return filepath.Join(home, ".local", "share", "crate")
}
