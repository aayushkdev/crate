package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var logsFollow bool

var logsCmd = &cobra.Command{
	Use:   "logs CONTAINER",
	Short: "Print container logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runtime.Logs(args[0], logsFollow, os.Stdout)
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	rootCmd.AddCommand(logsCmd)
}
