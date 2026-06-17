package main

import (
	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var execCmd = &cobra.Command{
	Use:   "exec CONTAINER COMMAND [ARG...]",
	Short: "Run a command in a running container",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runtime.Exec(args[0], args[1:])
	},
}

func init() {
	execCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(execCmd)
}
