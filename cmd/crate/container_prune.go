package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/container"
)

var containerPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove stopped containers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return container.Prune(os.Stdout, os.Stderr)
	},
}

func init() {
	containerCmd.AddCommand(containerPruneCmd)
}
