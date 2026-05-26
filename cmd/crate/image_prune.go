package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/image"
)

var imagePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove unreferenced image data",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return image.Prune(os.Stdout)
	},
}

func init() {
	imageCmd.AddCommand(imagePruneCmd)
}
