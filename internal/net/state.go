package net

import (
	"os"

	storage "github.com/aayushkdev/crate/internal/storage"
)

type State struct {
	Mode          string `json:"mode,omitempty"`
	Backend       string `json:"backend"`
	HelperPID     int    `json:"helper_pid"`
	InterfaceName string `json:"interface_name"`
	LogPath       string `json:"log_path,omitempty"`
}

func readState(containerID string) (*State, error) {
	var state State
	if err := storage.Read(storage.NetworkStatePath(containerID), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func writeState(containerID string, state *State) error {
	return storage.Write(storage.NetworkStatePath(containerID), state)
}

func removeState(containerID string) error {
	err := os.Remove(storage.NetworkStatePath(containerID))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
