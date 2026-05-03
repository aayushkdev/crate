package container

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	storage "github.com/aayushkdev/crate/internal/storage"
)

type Status string

const (
	StatusCreated  Status = "created"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
	StatusExited   Status = "exited"
	StatusStopped  Status = "stopped"
	StatusFailed   Status = "failed"
)

type State struct {
	ID          string    `json:"id"`
	Image       string    `json:"image"`
	Command     []string  `json:"command,omitempty"`
	Status      Status    `json:"status"`
	PID         int       `json:"pid,omitempty"`
	ExitCode    int       `json:"exit_code,omitempty"`
	LogPath     string    `json:"log_path,omitempty"`
	NetworkMode string    `json:"network_mode,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	FinishedAt  time.Time `json:"finished_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type Summary struct {
	ID       string
	Image    string
	Status   Status
	PID      int
	Command  string
	ExitCode int
}

func LogPath(id string) string {
	return storage.ContainerLogPath(id)
}

func writeState(id string, state *State) error {
	return storage.Write(storage.ContainerStatePath(id), state)
}

func ReadState(id string) (*State, error) {
	var state State
	err := storage.Read(storage.ContainerStatePath(id), &state)
	if err != nil {
		return nil, wrapNotFound(id, err)
	}

	return &state, nil
}

func UpdateState(id string, update func(*State)) error {
	state, err := ReadState(id)
	if err != nil {
		return err
	}

	update(state)
	return writeState(id, state)
}

func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	err := syscall.Kill(pid, 0)
	return err == nil
}

func RefreshState(id string) (*State, error) {
	state, err := ReadState(id)
	if err != nil {
		return nil, err
	}

	if state.Status == StatusRunning && !ProcessAlive(state.PID) {
		if state.FinishedAt.IsZero() {
			state.FinishedAt = time.Now().UTC()
		}
		state.Status = StatusExited
		if state.ExitCode == 0 {
			state.ExitCode = -1
		}
		if err := writeState(id, state); err != nil {
			return nil, err
		}
	}

	return state, nil
}

func ListSummaries() ([]Summary, error) {
	root := storage.ContainersDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	summaries := make([]Summary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		id := entry.Name()
		state, err := RefreshState(id)
		if err != nil {
			continue
		}

		summaries = append(summaries, Summary{
			ID:       state.ID,
			Image:    state.Image,
			Status:   state.Status,
			PID:      state.PID,
			Command:  strings.Join(state.Command, " "),
			ExitCode: state.ExitCode,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ID < summaries[j].ID
	})

	return summaries, nil
}

func FormatStatus(state *State) string {
	switch state.Status {
	case StatusExited:
		if state.ExitCode >= 0 {
			return fmt.Sprintf("%s (%d)", state.Status, state.ExitCode)
		}
		return string(state.Status)
	default:
		return string(state.Status)
	}
}
