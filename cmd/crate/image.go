package main

import "github.com/spf13/cobra"

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage images",
}

func init() {
	rootCmd.AddCommand(imageCmd)
}
