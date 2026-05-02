package runtime

import "github.com/aayushkdev/crate/internal/container"

func Run(image string, command []string, detach bool) (string, error) {
	containerID, err := container.Create(image)
	if err != nil {
		return "", err
	}

	if err := Start(containerID, command, detach); err != nil {
		return "", err
	}

	return containerID, nil
}
