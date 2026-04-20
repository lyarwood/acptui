package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print acptui version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("acptui %s\n", Version)
	},
}
