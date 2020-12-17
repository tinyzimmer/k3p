package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/version"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information for k3p",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("K3P Version:", version.K3pVersion)
		fmt.Println("K3P GitCommit:", version.K3pCommit)
	},
}
