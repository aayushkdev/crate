package runtime

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aayushkdev/crate/internal/container"
	cratenet "github.com/aayushkdev/crate/internal/net"
)

func PS(stdout io.Writer, all bool) error {
	summaries, err := container.ListSummaries()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if all {
		fmt.Fprintln(w, "CONTAINER\tNAME\tIMAGE\tSTATUS\tPID\tCREATED\tNETWORK\tPORTS\tCOMMAND")
	} else {
		fmt.Fprintln(w, "CONTAINER\tNAME\tIMAGE\tSTATUS\tPID\tCOMMAND")
	}
	for _, summary := range summaries {
		if !all && summary.Status != container.StatusRunning {
			continue
		}

		command := strings.TrimSpace(summary.Command)
		if command == "" {
			command = "-"
		}
		name := strings.TrimSpace(summary.Name)
		if name == "" {
			name = "-"
		}

		if all {
			fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				summary.ID,
				name,
				summary.Image,
				formatSummaryStatus(summary),
				formatPID(summary.PID),
				formatAge(summary.CreatedAt),
				formatNetwork(summary),
				formatPorts(summary.Network.Publish),
				command,
			)
		} else {
			fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				summary.ID,
				name,
				summary.Image,
				formatSummaryStatus(summary),
				formatPID(summary.PID),
				command,
			)
		}
	}

	return w.Flush()
}

func formatSummaryStatus(summary container.Summary) string {
	if summary.Status == container.StatusExited && summary.ExitCode != 0 {
		return fmt.Sprintf("%s (%d)", summary.Status, summary.ExitCode)
	}

	return string(summary.Status)
}

func formatPID(pid int) string {
	if pid <= 0 {
		return "-"
	}

	return fmt.Sprintf("%d", pid)
}

func formatNetwork(summary container.Summary) string {
	mode := cratenet.Mode(summary.NetworkMode)
	if mode == "" {
		mode = summary.Network.Mode
	}

	switch mode {
	case cratenet.ModeHost:
		return "host"
	case cratenet.ModeNone:
		return "none"
	case cratenet.ModePrivate:
		if summary.Network.Backend == "pasta" {
			return "pasta"
		}
		return "private"
	default:
		return "-"
	}
}

func formatPorts(ports []cratenet.PublishedPort) string {
	if len(ports) == 0 {
		return "-"
	}

	values := make([]string, 0, len(ports))
	for _, port := range ports {
		protocol := port.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		values = append(values, fmt.Sprintf("%d:%d/%s", port.HostPort, port.ContainerPort, protocol))
	}

	return strings.Join(values, ",")
}

func formatAge(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	elapsed := time.Since(t)
	if elapsed < 0 {
		elapsed = 0
	}

	switch {
	case elapsed < time.Minute:
		return plural(int(elapsed/time.Second), "second")
	case elapsed < time.Hour:
		return plural(int(elapsed/time.Minute), "minute")
	case elapsed < 24*time.Hour:
		return plural(int(elapsed/time.Hour), "hour")
	case elapsed < 7*24*time.Hour:
		return plural(int(elapsed/(24*time.Hour)), "day")
	case elapsed < 30*24*time.Hour:
		return plural(int(elapsed/(7*24*time.Hour)), "week")
	case elapsed < 365*24*time.Hour:
		return plural(int(elapsed/(30*24*time.Hour)), "month")
	default:
		return plural(int(elapsed/(365*24*time.Hour)), "year")
	}
}

func plural(value int, unit string) string {
	if value != 1 {
		unit += "s"
	}

	return fmt.Sprintf("%d %s ago", value, unit)
}
