package container

import (
	"fmt"

	cratenet "github.com/aayushkdev/crate/internal/net"
)

func Remove(id string) error {
	state, err := RefreshState(id)
	if err != nil {
		return err
	}

	if state.Status == StatusRunning || state.Status == StatusStopping {
		return fmt.Errorf("container %s is running; stop it first", id)
	}

	if err := cratenet.Teardown(id); err != nil {
		return err
	}

	if err := removeContainerDir(id); err != nil {
		return fmt.Errorf("remove container %s: %w", id, err)
	}

	return nil
}
