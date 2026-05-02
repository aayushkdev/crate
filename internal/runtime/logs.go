package runtime

import (
	"io"
	"os"
	"time"

	"github.com/aayushkdev/crate/internal/container"
)

func Logs(containerID string, follow bool, stdout io.Writer) error {
	logPath := container.LogPath(containerID)
	state, err := container.RefreshState(containerID)
	if err != nil {
		return err
	}

	var offset int64
	for {
		next, err := printLogs(logPath, offset, stdout)
		if err != nil {
			if os.IsNotExist(err) && !follow {
				return nil
			}
			return err
		}
		offset = next

		if !follow {
			return nil
		}

		state, err = container.RefreshState(containerID)
		if err != nil {
			return err
		}
		if state.Status != container.StatusRunning {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func printLogs(path string, offset int64, stdout io.Writer) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return offset, err
	}
	defer f.Close()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset, err
	}

	n, err := io.Copy(stdout, f)
	if err != nil {
		return offset, err
	}

	return offset + n, nil
}
