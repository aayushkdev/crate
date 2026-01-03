package main

import (
	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var runCmd = &cobra.Command{
	Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
	Short: "Run a command in a container",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]
		command := args[1:]

		return runtime.Run(image, command)
	},
}

func init() {
	runCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(runCmd)
}
