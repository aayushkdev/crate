package runtime

import (
	"errors"
	"log"
	"os/exec"
	"time"

	"github.com/aayushkdev/crate/internal/container"
	cratenet "github.com/aayushkdev/crate/internal/net"
)

func reapDetached(containerID string, cmd *exec.Cmd) {
	if err := finalizeContainer(containerID, cmd.Wait()); err != nil {
		log.Printf("crate: finalize detached container %s: %v", containerID, err)
	}
}

func finalizeContainer(containerID string, waitErr error) error {
	exitCode := exitCode(waitErr)
	status := container.StatusExited
	current, err := container.ReadState(containerID)
	if err == nil && current.Status == container.StatusStopping {
		status = container.StatusStopped
	}
	if err := container.UpdateState(containerID, func(s *container.State) {
		s.Status = status
		s.ExitCode = exitCode
		s.FinishedAt = time.Now().UTC()
	}); err != nil {
		return err
	}

	if err := cratenet.Teardown(containerID); err != nil {
		return err
	}
	warnPrivilegeDropFailure(containerID, exitCode)

	return waitErr
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return 1
}
