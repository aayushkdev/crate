package state

import (
	"path/filepath"

	"github.com/aayushkdev/crate/internal/image"
)

func ContainerDir(id string) string {
	return filepath.Join(image.CrateRoot(), "containers", id)
}

func ContainerRootfsPath(id string) string {
	return filepath.Join(ContainerDir(id), "rootfs")
}

func ContainerConfigPath(id string) string {
	return filepath.Join(ContainerDir(id), "config.json")
}

func ContainerStatePath(id string) string {
	return filepath.Join(ContainerDir(id), "state.json")
}

func ContainerLogPath(id string) string {
	return filepath.Join(ContainerDir(id), "logs", "container.log")
}

func NetworkStatePath(id string) string {
	return filepath.Join(ContainerDir(id), "network.json")
}

func NetworkLogPath(id string) string {
	return filepath.Join(ContainerDir(id), "logs", "network.log")
}
