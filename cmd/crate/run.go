package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
	"github.com/aayushkdev/crate/internal/runtime"
)

var runDetach bool
var runUser string

var runCmd = &cobra.Command{
	Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
	Short: "Run a command in a container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]
		command := args[1:]

		containerID, err := runtime.Run(image, command, runDetach, container.CreateOptions{
			User: runUser,
		})
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
	runCmd.Flags().StringVar(&runUser, "user", "", "Run the container as a specific user (USER, UID, USER:GROUP, or UID:GID)")
	rootCmd.AddCommand(runCmd)
}
