package cmd

import (
	"log"
	"os"
	"path"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/bundle"
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

	buildCmd.Flags().StringVarP(&buildK3sVersion, "version", "V", bundle.VersionLatest, "The k3s version to bundle with the package")
	buildCmd.Flags().StringVarP(&buildManifestDir, "manifests", "m", cwd, "The directory to scan for kubernetes manifests, defaults to the current directory")
	buildCmd.Flags().StringSliceVarP(&buildExcludeDirs, "exclude", "e", []string{}, "Directories to exclude when reading the manifest directory")
	buildCmd.Flags().StringVarP(&buildArch, "arch", "a", "amd64", "The architecture to package the distribution for")
	buildCmd.Flags().StringVarP(&buildOutput, "output", "o", path.Join(cwd, "package.tar"), "The file to save the distribution package to")

	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build an embedded k3s distribution package",
	RunE:  build,
}

func build(cmd *cobra.Command, args []string) error {
	builder := bundle.NewBuilder(buildK3sVersion, buildArch)
	if err := builder.Setup(); err != nil {
		return err
	}
	return builder.Build(&bundle.BuildOptions{
		ManifestDir: buildManifestDir,
		Excludes:    buildExcludeDirs,
		Output:      buildOutput,
	})
}
