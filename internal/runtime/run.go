package runtime

import "github.com/aayushkdev/crate/internal/container"

func Run(image string, command []string, detach bool, opts container.CreateOptions, workingDir string) (string, error) {
	containerID, err := container.Create(image, opts)
	if err != nil {
		return "", err
	}

	if err := Start(containerID, command, detach, "", workingDir); err != nil {
		return "", err
	}

	return containerID, nil
}
