package net

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/aayushkdev/crate/internal/image"
)

type State struct {
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
	f, err := os.Open(statePath(containerID))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var state State
	if err := json.NewDecoder(f).Decode(&state); err != nil {
		return nil, err
	}

	return &state, nil
}

func writeState(containerID string, state *State) error {
	path := statePath(containerID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(state); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}
