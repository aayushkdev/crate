package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
	cratenet "github.com/aayushkdev/crate/internal/net"
	"github.com/aayushkdev/crate/internal/runtime"
)

var runDetach bool
var runPublish []string
var runNetwork string
var runUser string
var runName string

var runCmd = &cobra.Command{
	Use:   "run [OPTIONS] IMAGE [COMMAND] [ARG...]",
	Short: "Run a command in a container",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]
		command := args[1:]

		publish, err := cratenet.ParsePublishedPorts(runPublish)
		if err != nil {
			return err
		}
		networkMode, err := cratenet.ParseMode(runNetwork)
		if err != nil {
			return err
		}

		containerID, err := runtime.Run(image, command, runDetach, container.CreateOptions{
			Publish:     publish,
			NetworkMode: networkMode,
			User:        runUser,
			Name:        runName,
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
	runCmd.Flags().StringArrayVarP(&runPublish, "publish", "p", nil, "Publish a container port to the host (HOST:CONTAINER[/tcp|udp])")
	runCmd.Flags().StringVarP(&runNetwork, "network", "n", "", "Network mode: host, none, or private")
	runCmd.Flags().StringVar(&runUser, "user", "", "Run the container as a specific user (USER, UID, USER:GROUP, or UID:GID)")
	runCmd.Flags().StringVar(&runName, "name", "", "Assign a name to the container")
	rootCmd.AddCommand(runCmd)
}
