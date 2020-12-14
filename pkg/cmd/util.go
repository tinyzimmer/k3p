package cmd

import (
	"github.com/spf13/cobra"
)

func completeStringOpts(opts []string) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return opts, cobra.ShellCompDirectiveDefault
	}
}
