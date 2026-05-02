package runtime

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/aayushkdev/crate/internal/container"
)

func PS(stdout io.Writer, all bool) error {
	summaries, err := container.ListSummaries()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CONTAINER\tIMAGE\tSTATUS\tPID\tCOMMAND")
	for _, summary := range summaries {
		if !all && summary.Status != container.StatusRunning {
			continue
		}

		command := strings.TrimSpace(summary.Command)
		if command == "" {
			command = "-"
		}

		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%d\t%s\n",
			summary.ID,
			summary.Image,
			formatSummaryStatus(summary),
			summary.PID,
			command,
		)
	}

	return w.Flush()
}

func formatSummaryStatus(summary container.Summary) string {
	if summary.Status == container.StatusExited && summary.ExitCode != 0 {
		return fmt.Sprintf("%s (%d)", summary.Status, summary.ExitCode)
	}

	return string(summary.Status)
}
