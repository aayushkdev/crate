package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
)

var createCmd = &cobra.Command{
	Use:   "create IMAGE",
	Short: "Create a container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		id, err := container.Create(image)
		if err != nil {
			return err
		}

		fmt.Println(id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
