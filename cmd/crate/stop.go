package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var stopCmd = &cobra.Command{
	Use:   "stop CONTAINER [CONTAINER...]",
	Short: "Stop one or more running containers",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, containerID := range args {
			if err := runtime.Stop(containerID); err != nil {
				return err
			}
			fmt.Println(containerID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
