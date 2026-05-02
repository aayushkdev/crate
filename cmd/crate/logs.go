package main

import (
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
)

var logsFollow bool

var logsCmd = &cobra.Command{
	Use:   "logs CONTAINER",
	Short: "Print container logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		logPath := container.LogPath(containerID)

		var offset int64
		for {
			next, err := printLogs(logPath, offset)
			if err != nil {
				return err
			}
			offset = next

			if !logsFollow {
				return nil
			}

			state, err := container.RefreshState(containerID)
			if err != nil {
				return err
			}
			if state.Status != container.StatusRunning {
				return nil
			}

			time.Sleep(500 * time.Millisecond)
		}
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	rootCmd.AddCommand(logsCmd)
}

func printLogs(path string, offset int64) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return offset, err
	}
	defer f.Close()

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return offset, err
	}

	n, err := io.Copy(os.Stdout, f)
	if err != nil {
		return offset, err
	}

	return offset + n, nil
}
