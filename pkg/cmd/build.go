package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/build"
	"github.com/tinyzimmer/k3p/pkg/cache"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var (
	buildPullPolicy string
	buildOpts       *types.BuildOptions
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var defaultConfig string
	if _, err := os.Stat(path.Join(cwd, "k3p.yaml")); err == nil {
		defaultConfig = path.Join(cwd, "k3p.yaml")
	}

	buildOpts = &types.BuildOptions{}

	buildCmd.Flags().StringVarP(&buildOpts.Name, "name", "n", "", "The name to give the package, if not provided one will be generated")
	buildCmd.Flags().StringVarP(&buildOpts.BuildVersion, "version", "V", types.VersionLatest, "The version to tag the package")
	buildCmd.Flags().StringVar(&buildOpts.K3sVersion, "k3s-version", types.VersionLatest, "A specific k3s version to bundle with the package, overrides --channel")
	buildCmd.Flags().StringVarP(&buildOpts.K3sChannel, "channel", "C", "stable", "The release channel to retrieve the version of k3s from")
	buildCmd.Flags().StringArrayVarP(&buildOpts.ManifestDirs, "manifests", "m", []string{cwd}, "Directories to scan for kubernetes manifests and charts, defaults to the current directory, can be specified multiple times")
	buildCmd.Flags().StringSliceVarP(&buildOpts.Excludes, "exclude", "e", []string{}, "Directories to exclude when reading the manifest directory")
	buildCmd.Flags().StringVarP(&buildOpts.Arch, "arch", "a", runtime.GOARCH, "The architecture to package the distribution for. Only (amd64, arm, and arm64 are supported)")
	buildCmd.Flags().StringVarP(&buildOpts.ImageFile, "image-file", "I", "", "A file containing a list of extra images to bundle with the archive")
	buildCmd.Flags().StringSliceVarP(&buildOpts.Images, "images", "i", []string{}, "A comma separated list of images to include with the archive")
	buildCmd.Flags().StringVarP(&buildOpts.EULAFile, "eula", "E", "", "A file containing an End User License Agreement to display to the user upon installing the package")
	buildCmd.Flags().StringVarP(&buildOpts.Output, "output", "o", path.Join(cwd, "package.tar"), "The file to save the distribution package to")
	buildCmd.Flags().BoolVar(&buildOpts.ExcludeImages, "exclude-images", false, "Don't include container images with the final archive")
	buildCmd.Flags().StringVar(&buildPullPolicy, "pull-policy", string(types.PullPolicyAlways), "The pull policy to use when bundling container images (valid options always,never,ifnotpresent [case-insensitive])")
	buildCmd.Flags().StringVarP(&buildOpts.ConfigFile, "config", "c", defaultConfig, "An optional file providing variables and other configurations to be used at installation, if a k3p.yaml in the current directory exists it will be used automatically")
	buildCmd.Flags().BoolVarP(&cache.NoCache, "no-cache", "N", false, "Disable the use of the local cache when downloading assets")
	buildCmd.Flags().BoolVar(&buildOpts.Compress, "compress", false, "Whether to apply zst encryption to the package, it will usually require the same k3p release to decompress.")

	buildCmd.MarkFlagDirname("exclude")
	buildCmd.MarkFlagDirname("manifests")
	buildCmd.MarkFlagFilename("config", "json", "yaml", "yml")
	buildCmd.RegisterFlagCompletionFunc("pull-policy", completeStringOpts([]string{string(types.PullPolicyAlways), string(types.PullPolicyIfNotPresent), string(types.PullPolicyNever)}))
	buildCmd.RegisterFlagCompletionFunc("arch", completeStringOpts([]string{"amd64", "arm64", "arm"}))
	buildCmd.RegisterFlagCompletionFunc("channel", completeChannels)

	rootCmd.AddCommand(buildCmd)
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a k3s distribution package",
	RunE: func(cmd *cobra.Command, args []string) error {
		// validate pull policy first
		switch types.PullPolicy(strings.ToLower(buildPullPolicy)) {
		case types.PullPolicyAlways:
			buildOpts.PullPolicy = types.PullPolicyAlways
		case types.PullPolicyNever:
			buildOpts.PullPolicy = types.PullPolicyNever
		case types.PullPolicyIfNotPresent:
			buildOpts.PullPolicy = types.PullPolicyIfNotPresent
		default:
			return fmt.Errorf("%s is not a valid pull policy", buildPullPolicy)
		}
		builder, err := build.NewBuilder()
		if err != nil {
			return err
		}
		return builder.Build(buildOpts)
	},
}

type channelResponse struct {
	Data []channel `json:"data"`
}

type channel struct {
	ID string `json:"id"`
}

func completeChannels(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log.Verbose = false
	var res channelResponse
	resp, err := cache.DefaultCache.GetIfOlder("https://update.k3s.io/v1-release/channels", time.Hour*24)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer resp.Close()
	body, err := ioutil.ReadAll(resp)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, len(res.Data))
	for i, channel := range res.Data {
		out[i] = channel.ID
	}
	return out, cobra.ShellCompDirectiveDefault
}
