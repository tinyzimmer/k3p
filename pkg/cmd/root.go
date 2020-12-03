package cmd

import (
	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/cache"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/util"
)

var (
	cacheDir string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", cache.DefaultCache.CacheDir(), "Override the default location for cached k3s assets")
	rootCmd.PersistentFlags().StringVar(&util.TempDir, "tmp-dir", util.TempDir, "Override the default tmp directory")
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
			log.Debugf("Setting cache dir to %q\n", cacheDir)
			cache.DefaultCache = cache.New(cacheDir)
		} else {
			log.Debugf("Default cache dir is %q\n", cache.DefaultCache.CacheDir())
		}
	},
}

// GetRootCommand returns the root k3p command
func GetRootCommand() *cobra.Command { return rootCmd }
