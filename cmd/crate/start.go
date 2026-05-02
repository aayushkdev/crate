package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var startDetach bool

var startCmd = &cobra.Command{
	Use:   "start CONTAINER [COMMAND] [ARG...]",
	Short: "Start an existing container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		containerID := args[0]
		command := args[1:]

		if err := runtime.Start(containerID, command, startDetach); err != nil {
			return err
		}

		if startDetach {
			fmt.Println(containerID)
		}

		return nil
	},
}

func init() {
	startCmd.Flags().SetInterspersed(false)
	startCmd.Flags().BoolVarP(&startDetach, "detach", "d", false, "Run the container in the background")
	rootCmd.AddCommand(startCmd)
}
