package main

import (
	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var runCmd = &cobra.Command{
	Use:   "run [COMMAND...]",
	Short: "Run a command in a container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runtime.LaunchContainer(args)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
