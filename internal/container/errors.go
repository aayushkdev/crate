package container

import (
	"fmt"
	"os"
)

func notFoundError(id string) error {
	return fmt.Errorf("container %s not found", id)
}

func wrapNotFound(id string, err error) error {
	if os.IsNotExist(err) {
		return notFoundError(id)
	}

	return err
}
