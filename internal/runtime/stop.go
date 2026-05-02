package runtime

import (
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/aayushkdev/crate/internal/container"
)

func Stop(containerID string) error {
	state, err := container.RefreshState(containerID)
	if err != nil {
		return err
	}

	if state.Status != container.StatusRunning && state.Status != container.StatusStopping {
		return fmt.Errorf("container %s is not running", containerID)
	}

	if err := container.UpdateState(containerID, func(s *container.State) {
		s.Status = container.StatusStopping
	}); err != nil {
		return err
	}

	if err := killProcessGroup(state.PID, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}

	if waitForExit(state.PID, 1*time.Second) {
		return container.UpdateState(containerID, func(s *container.State) {
			s.Status = container.StatusStopped
			s.FinishedAt = time.Now().UTC()
		})
	}

	if err := killProcessGroup(state.PID, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}

	for !waitForExit(state.PID, 100*time.Millisecond) {
	}

	return container.UpdateState(containerID, func(s *container.State) {
		s.Status = container.StatusStopped
		s.FinishedAt = time.Now().UTC()
	})
}
