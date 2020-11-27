package cmd

import (
	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/cache"
)

func init() {
	cacheCmd.AddCommand(cacheCleanCmd)
	rootCmd.AddCommand(cacheCmd)
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Cache management options",
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Wipe the local artifact cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cache.DefaultCache.Clean()
	},
}
