package container

import (
	"fmt"
	"io"
	"os"
)

func Prune(stdout io.Writer, stderr io.Writer) error {
	summaries, err := ListSummaries()
	if err != nil {
		return err
	}

	removed := 0
	for _, summary := range summaries {
		if summary.Status == StatusRunning || summary.Status == StatusStopping {
			continue
		}
		cfg, err := ReadConfig(summary.ID)
		if err != nil {
			return err
		}
		if !cfg.Rootless && os.Geteuid() != 0 {
			fmt.Fprintf(stderr, "crate: warning: skipping rootful container %s; run crate as root to prune it\n", summary.ID)
			continue
		}
		if err := Remove(summary.ID); err != nil {
			return err
		}
		fmt.Fprintln(stdout, summary.ID)
		removed++
	}

	fmt.Fprintf(stdout, "Removed containers: %d\n", removed)
	return nil
}
