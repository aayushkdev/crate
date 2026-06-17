package main

import (
	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var execCmd = &cobra.Command{
	Use:   "exec CONTAINER [COMMAND] [ARG...]",
	Short: "Run a command in a running container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := args[1:]
		if len(command) == 0 {
			command = []string{"sh"}
		}

		return runtime.Exec(args[0], command)
	},
}

func init() {
	execCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(execCmd)
}
