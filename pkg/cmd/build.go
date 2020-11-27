package cmd

import (
	"log"
	"os"
	"path"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/build"
	"github.com/tinyzimmer/k3p/pkg/cache"
)

var (
	buildK3sVersion  string
	buildManifestDir string
	buildExcludeDirs []string
	buildArch        string
	buildOutput      string
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	buildCmd.Flags().StringVarP(&buildK3sVersion, "version", "V", build.VersionLatest, "The k3s version to bundle with the package")
	buildCmd.Flags().StringVarP(&buildManifestDir, "manifests", "m", cwd, "The directory to scan for kubernetes manifests, defaults to the current directory")
	buildCmd.Flags().StringSliceVarP(&buildExcludeDirs, "exclude", "e", []string{}, "Directories to exclude when reading the manifest directory")
	buildCmd.Flags().StringVarP(&buildArch, "arch", "a", runtime.GOARCH, "The architecture to package the distribution for. Only (amd64, arm, and arm64 are supported)")
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", path.Join(cwd, "package.tar"), "The file to save the distribution package to")
	buildCmd.Flags().BoolVarP(&cache.NoCache, "no-cache", "N", false, "Disable the use of the local cache when downloading assets.")

	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build an embedded k3s distribution package",
	RunE: func(cmd *cobra.Command, args []string) error {
		builder := build.NewBuilder(buildK3sVersion, buildArch)
		if err := builder.Setup(); err != nil {
			return err
		}
		return builder.Build(&build.Options{
			ManifestDir: buildManifestDir,
			Excludes:    buildExcludeDirs,
			Output:      buildOutput,
		})
	},
}
