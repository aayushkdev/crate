package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var runDetach bool

var runCmd = &cobra.Command{
	Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
	Short: "Run a command in a container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]
		command := args[1:]

		containerID, err := runtime.Run(image, command, runDetach)
		if err != nil {
			return err
		}

		if runDetach {
			fmt.Println(containerID)
		}

		return nil
	},
}

func init() {
	runCmd.Flags().SetInterspersed(false)
	runCmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "Run the container in the background")
	rootCmd.AddCommand(runCmd)
}
