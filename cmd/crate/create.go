package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
)

var createPublish []string
var createUser string
var createName string

var createCmd = &cobra.Command{
	Use:   "create IMAGE",
	Short: "Create a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		publish, err := parsePublishedPorts(createPublish)
		if err != nil {
			return err
		}

		id, err := container.Create(image, container.CreateOptions{
			Publish: publish,
			User:    createUser,
			Name:    createName,
		})
		if err != nil {
			return err
		}

		fmt.Println(id)
		return nil
	},
}

func init() {
	createCmd.Flags().StringArrayVarP(&createPublish, "publish", "p", nil, "Publish a container port to the host (HOST:CONTAINER[/tcp|udp])")
	createCmd.Flags().StringVar(&createUser, "user", "", "Run the container as a specific user (USER, UID, USER:GROUP, or UID:GID)")
	createCmd.Flags().StringVar(&createName, "name", "", "Assign a name to the container")
	rootCmd.AddCommand(createCmd)
}
