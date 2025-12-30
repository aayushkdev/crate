package main

import "github.com/spf13/cobra"

// Execute runs the root Cobra command.
func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "crate",
	Short: "crate is a lightweight container runtime",
}

func init() {
	// Avoid printing usage or Cobra-managed errors for child process exits.
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
}
