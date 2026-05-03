package state

import "os"

type Network struct {
	Mode          string `json:"mode,omitempty"`
	Backend       string `json:"backend"`
	HelperPID     int    `json:"helper_pid"`
	InterfaceName string `json:"interface_name"`
	LogPath       string `json:"log_path,omitempty"`
}

func ReadNetwork(id string) (*Network, error) {
	var network Network
	if err := Read(NetworkStatePath(id), &network); err != nil {
		return nil, err
	}

	return &network, nil
}

func WriteNetwork(id string, network *Network) error {
	return Write(NetworkStatePath(id), network)
}

func RemoveNetwork(id string) error {
	err := os.Remove(NetworkStatePath(id))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
