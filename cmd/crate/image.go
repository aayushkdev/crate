package main

import (
	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/image"
)

var pullCmd = &cobra.Command{
	Use:   "pull IMAGE",
	Short: "Pull an image from a registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref := args[0]
		return image.Pull(ref)
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
