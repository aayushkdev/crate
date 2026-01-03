package main

import (
	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var startCmd = &cobra.Command{
	Use:   "start CONTAINER [COMMAND] [ARG...]",
	Short: "Start an existing container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		command := args[1:]

		return runtime.Start(containerID, command)
	},
}

func init() {
	startCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(startCmd)
}
