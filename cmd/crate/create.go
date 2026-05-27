package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
	cratenet "github.com/aayushkdev/crate/internal/net"
)

var createPublish []string
var createNetwork string
var createEnv []string
var createUser string
var createName string
var createMounts []string

var createCmd = &cobra.Command{
	Use:   "create IMAGE",
	Short: "Create a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		publish, err := cratenet.ParsePublishedPorts(createPublish)
		if err != nil {
			return err
		}
		networkMode, err := cratenet.ParseMode(createNetwork)
		if err != nil {
			return err
		}
		mounts, err := container.ParseMounts(createMounts)
		if err != nil {
			return err
		}

		id, warnings, err := container.Create(image, container.CreateOptions{
			Publish:     publish,
			NetworkMode: networkMode,
			Env:         createEnv,
			User:        createUser,
			Name:        createName,
			Mounts:      mounts,
		})
		if err != nil {
			return err
		}
		for _, warning := range warnings {
			fmt.Fprintf(cmd.ErrOrStderr(), "crate: warning: %s\n", warning)
		}

		fmt.Println(id)
		return nil
	},
}

func init() {
	createCmd.Flags().StringArrayVarP(&createPublish, "publish", "p", nil, "Publish a container port to the host (HOST:CONTAINER[/tcp|udp])")
	createCmd.Flags().StringVarP(&createNetwork, "network", "n", "", "Network mode: host, none, or private")
	createCmd.Flags().StringArrayVarP(&createEnv, "env", "e", nil, "Set environment variable (KEY=value)")
	createCmd.Flags().StringVar(&createUser, "user", "", "Run the container as a specific user (USER, UID, USER:GROUP, or UID:GID)")
	createCmd.Flags().StringVar(&createName, "name", "", "Assign a name to the container")
	createCmd.Flags().StringArrayVarP(&createMounts, "volume", "v", nil, "Bind mount a host path (HOST_PATH:CONTAINER_PATH[:ro])")
	rootCmd.AddCommand(createCmd)
}
