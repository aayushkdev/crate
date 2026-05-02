package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/image"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List images",
	RunE: func(cmd *cobra.Command, args []string) error {
		return image.Images(os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
}
