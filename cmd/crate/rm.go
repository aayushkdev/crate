package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
)

var rmCmd = &cobra.Command{
	Use:   "rm CONTAINER [CONTAINER...]",
	Short: "Remove one or more containers",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, containerID := range args {
			if err := container.Remove(containerID); err != nil {
				return err
			}
			fmt.Println(containerID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
