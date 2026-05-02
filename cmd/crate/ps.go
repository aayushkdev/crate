package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/aayushkdev/crate/internal/runtime"
)

var psAll bool

var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "List containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runtime.PS(os.Stdout, psAll)
	},
}

func init() {
	psCmd.Flags().BoolVarP(&psAll, "all", "a", false, "Show all containers")
	rootCmd.AddCommand(psCmd)
}
