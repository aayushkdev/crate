package storage

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	containersDirName   = "containers"
	imagesDirName       = "images"
	blobsDirName        = "blobs"
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

func ImagesDir() string {
	return filepath.Join(CrateRoot(), imagesDirName)
}

func ImageMetadataPath(digest string) (string, error) {
	if digest == "" {
		return "", fmt.Errorf("invalid digest: %s", digest)
	}

	return filepath.Join(ImagesDir(), digest), nil
}

func BlobPath(digest string) (string, error) {
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid digest: %s", digest)
	}

	algo, hash := parts[0], parts[1]
	return filepath.Join(CrateRoot(), blobsDirName, algo, hash), nil
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
