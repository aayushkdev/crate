package runtime

import "github.com/aayushkdev/crate/internal/container"

func Start(containerID string, command []string, detach bool, user string) error {
	cfg, err := container.ReadConfig(containerID)
	if err != nil {
		return err
	}
	containerID = cfg.ID
	if user != "" {
		cfg.User = user
	}

	if detach {
		return launchContainer(containerID, command, cfg, false, false)
	}

	return launchContainer(containerID, command, cfg, true, true)
}
