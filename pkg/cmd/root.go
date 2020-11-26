package cmd

import "github.com/spf13/cobra"

var (
	verbose bool
)

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}

var rootCmd = &cobra.Command{
	Use:   "k3p",
	Short: "k3p is a k3s packaging and delivery utility",
	Long: `
The k3p command provides an easy method for packaging a kubernetes environment into a distributable object.
`,
	SilenceUsage:      true,
	DisableAutoGenTag: true,
}

// GetRootCommand returns the root k3p command
func GetRootCommand() *cobra.Command { return rootCmd }
