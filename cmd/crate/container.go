package main

import "github.com/spf13/cobra"

var containerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage containers",
}

func init() {
	rootCmd.AddCommand(containerCmd)
}
