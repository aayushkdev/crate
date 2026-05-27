package runtime

import (
	"errors"

	"github.com/aayushkdev/crate/internal/container"
)

func Run(image string, command []string, detach bool, opts container.CreateOptions, workingDir string) (string, []string, error) {
	containerID, warnings, err := container.Create(image, opts)
	if err != nil {
		return "", nil, err
	}

	if err := Start(containerID, command, detach, "", workingDir); err != nil {
		if cleanupErr := cleanupFailedRun(containerID); cleanupErr != nil {
			return "", nil, errors.Join(err, cleanupErr)
		}
		return "", nil, err
	}

	return containerID, warnings, nil
}

func cleanupFailedRun(containerID string) error {
	if err := container.Remove(containerID); err == nil {
		return nil
	}

	if err := Stop(containerID); err != nil {
		if removeErr := container.Remove(containerID); removeErr != nil {
			return errors.Join(err, removeErr)
		}
		return nil
	}

	return container.Remove(containerID)
}
