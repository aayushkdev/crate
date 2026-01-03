package runtime

import (
	"github.com/aayushkdev/crate/internal/container"
)

func Run(image string, command []string) error {
	containerID, err := container.Create(image)
	if err != nil {
		return err
	}

	return Start(containerID, command)
}
