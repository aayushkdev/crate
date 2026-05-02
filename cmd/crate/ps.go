package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
)

var psAll bool

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		summaries, err := container.ListSummaries()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CONTAINER\tIMAGE\tSTATUS\tPID\tCOMMAND")
		for _, summary := range summaries {
			if !psAll && summary.Status != container.StatusRunning {
				continue
			}

			status := string(summary.Status)
			if summary.Status == container.StatusExited && summary.ExitCode != 0 {
				status = fmt.Sprintf("%s (%d)", status, summary.ExitCode)
			}
			command := strings.TrimSpace(summary.Command)
			if command == "" {
				command = "-"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", summary.ID, summary.Image, status, summary.PID, command)
		}

		return w.Flush()
	},
}

func init() {
	psCmd.Flags().BoolVarP(&psAll, "all", "a", false, "Show all containers")
	rootCmd.AddCommand(psCmd)
}
