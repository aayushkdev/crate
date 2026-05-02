package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/image"
)

var rmiCmd = &cobra.Command{
	Use:   "rmi IMAGE [IMAGE...]",
	Short: "Remove one or more images",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, imageRef := range args {
			if err := image.Remove(imageRef); err != nil {
				return err
			}
			fmt.Println(imageRef)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmiCmd)
}
