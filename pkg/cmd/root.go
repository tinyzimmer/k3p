package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/cache"
	"github.com/tinyzimmer/k3p/pkg/log"
)

var (
	cacheDir, tmpDir string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cacheDir, "cache", cache.DefaultCache.CacheDir(), "Override the default location for cached k3s assets")
	rootCmd.PersistentFlags().StringVar(&tmpDir, "tmp-dir", os.TempDir(), "Override the default tmp directory")
	rootCmd.PersistentFlags().BoolVarP(&log.Verbose, "verbose", "v", false, "Enable verbose logging")
}

var rootCmd = &cobra.Command{
	Use:   "k3p",
	Short: "k3p is a k3s packaging and delivery utility",
	Long: `
The k3p command provides an easy method for packaging a kubernetes environment into a distributable object.
`,
	SilenceUsage:      true,
	DisableAutoGenTag: true,
	SilenceErrors:     true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if cacheDir != cache.DefaultCache.CacheDir() {
			log.Debugf("Setting cache dir to %q", cacheDir)
			cache.DefaultCache = cache.New(cacheDir)
		} else {
			log.Debugf("Default cache dir is %q", cache.DefaultCache.CacheDir())
		}
	},
}

// GetRootCommand returns the root k3p command
func GetRootCommand() *cobra.Command { return rootCmd }
