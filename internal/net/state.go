package net

import (
	"path/filepath"

	"github.com/aayushkdev/crate/internal/image"
	filestate "github.com/aayushkdev/crate/internal/state"
)

type State struct {
	Mode          Mode   `json:"mode,omitempty"`
	Backend       string `json:"backend"`
	HelperPID     int    `json:"helper_pid"`
	InterfaceName string `json:"interface_name"`
	LogPath       string `json:"log_path,omitempty"`
}

func containerDir(containerID string) string {
	return filepath.Join(image.CrateRoot(), "containers", containerID)
}

func rootfsPath(containerID string) string {
	return filepath.Join(containerDir(containerID), "rootfs")
}

func statePath(containerID string) string {
	return filepath.Join(containerDir(containerID), "network.json")
}

func logPath(containerID string) string {
	return filepath.Join(containerDir(containerID), "logs", "network.log")
}

func readState(containerID string) (*State, error) {
	var state State
	if err := filestate.Read(statePath(containerID), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func writeState(containerID string, state *State) error {
	return filestate.Write(statePath(containerID), state)
}
