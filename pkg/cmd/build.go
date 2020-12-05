package cmd

import (
	"log"
	"os"
	"path"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/build"
	"github.com/tinyzimmer/k3p/pkg/cache"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var (
	buildOpts *types.BuildOptions
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	buildOpts = &types.BuildOptions{}

	buildCmd.Flags().StringVarP(&buildOpts.Name, "name", "n", "", "The name to give the package")
	buildCmd.Flags().StringVarP(&buildOpts.BuildVersion, "version", "V", types.VersionLatest, "The version to tag the package")
	buildCmd.Flags().StringVar(&buildOpts.K3sVersion, "k3s-version", types.VersionLatest, "A specific k3s version to bundle with the package, overrides --channel")
	buildCmd.Flags().StringVarP(&buildOpts.K3sChannel, "channel", "c", "stable", "The release channel to retrieve the version of k3s from")
	buildCmd.Flags().StringVarP(&buildOpts.ManifestDir, "manifests", "m", cwd, "The directory to scan for kubernetes manifests and charts, defaults to the current directory")
	buildCmd.Flags().StringVarP(&buildOpts.HelmArgs, "helm-args", "H", "", "Arguments to pass to the 'helm template' command when searching for images")
	buildCmd.Flags().StringSliceVarP(&buildOpts.Excludes, "exclude", "e", []string{}, "Directories to exclude when reading the manifest directory")
	buildCmd.Flags().StringVarP(&buildOpts.Arch, "arch", "a", runtime.GOARCH, "The architecture to package the distribution for. Only (amd64, arm, and arm64 are supported)")
	buildCmd.Flags().StringVarP(&buildOpts.ImageFile, "images", "i", "", "A file containing a list of extra images to bundle with the archive")
	buildCmd.Flags().StringVarP(&buildOpts.EULAFile, "eula", "E", "", "A file containing an End User License Agreement to display to the user upon installing the package")
	buildCmd.Flags().StringVarP(&buildOpts.Output, "output", "o", path.Join(cwd, "package.tar"), "The file to save the distribution package to")
	buildCmd.Flags().BoolVarP(&cache.NoCache, "no-cache", "N", false, "Disable the use of the local cache when downloading assets.")

	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build an embedded k3s distribution package",
	RunE: func(cmd *cobra.Command, args []string) error {
		builder, err := build.NewBuilder()
		if err != nil {
			return err
		}
		return builder.Build(buildOpts)
	},
}
