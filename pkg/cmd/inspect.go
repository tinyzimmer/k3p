package cmd

import (
	"archive/tar"
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/tinyzimmer/k3p/pkg/build/package/v1"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var inspectDetails bool
var inspectManifest string

func init() {
	inspectCmd.Flags().BoolVarP(&inspectDetails, "details", "D", false, "Show additional details on package content")
	inspectCmd.Flags().StringVarP(&inspectManifest, "manifest", "m", "", "Dump the contents of the specified manifest")

	inspectCmd.RegisterFlagCompletionFunc("manifest", completeManifests)

	rootCmd.AddCommand(inspectCmd)
}

func completeManifests(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log.Verbose = false

	var pkgReader io.ReadCloser
	var err error

	pkgReader, err = os.Open(args[0])
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if strings.HasSuffix(args[0], ".zst") {
		pkgReader, err = v1.Decompress(pkgReader)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
	}

	pkg, err := v1.Load(pkgReader)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer pkg.Close()
	manifest := pkg.GetMeta().GetManifest()
	return manifest.K8sManifests, cobra.ShellCompDirectiveDefault
}

var inspectCmd = &cobra.Command{
	Use:   "inspect PACKAGE",
	Short: "Inspect the given package",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"tar"}, cobra.ShellCompDirectiveFilterFileExt
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var pkgReader io.ReadCloser
		var err error

		pkgReader, err = os.Open(args[0])
		if err != nil {
			return err
		}

		if strings.HasSuffix(args[0], ".zst") {
			pkgReader, err = v1.Decompress(pkgReader)
			if err != nil {
				return err
			}
		}

		pkg, err := v1.Load(pkgReader)
		if err != nil {
			return err
		}
		defer pkg.Close()

		meta := pkg.GetMeta()

		if inspectManifest != "" {
			artifact := &types.Artifact{
				Type: types.ArtifactManifest,
				Name: inspectManifest,
			}
			if err := pkg.Get(artifact); err != nil {
				return err
			}
			defer artifact.Body.Close()
			body, err := ioutil.ReadAll(artifact.Body)
			if err != nil {
				return err
			}
			fmt.Println(string(body))
			return nil
		}

		fmt.Println()
		fmt.Println("NAME:   ", meta.Name)
		fmt.Println("VERSION:", meta.Version)
		fmt.Println()
		fmt.Println("ARCH:       ", meta.Arch)
		fmt.Println("K3S VERSION:", meta.K3sVersion)

		fmt.Println()
		fmt.Println("CONTENTS:")

		fmt.Println()
		fmt.Println("  BINARIES")
		for _, bin := range meta.Manifest.Bins {
			artifact := &types.Artifact{Type: types.ArtifactBin, Name: bin}
			if err := pkg.Get(artifact); err != nil {
				return err
			}
			fmt.Println("    ", artifact.Name, "\t", byteCountSI(artifact.Size))
		}

		fmt.Println()
		fmt.Println("  SCRIPTS")
		for _, sc := range meta.Manifest.Scripts {
			artifact := &types.Artifact{Type: types.ArtifactScript, Name: sc}
			if err := pkg.Get(artifact); err != nil {
				return err
			}
			fmt.Println("    ", artifact.Name, "\t", byteCountSI(artifact.Size))
		}

		fmt.Println()
		fmt.Println("  CONFIGS")
		for _, e := range meta.Manifest.Etc {
			artifact := &types.Artifact{Type: types.ArtifactEtc, Name: e}
			if err := pkg.Get(artifact); err != nil {
				return err
			}
			fmt.Println("    ", artifact.Name, "\t", byteCountSI(artifact.Size))
		}

		fmt.Println()
		fmt.Println("  IMAGES")
		if inspectDetails {
			fmt.Println()
		}
		for i, img := range meta.Manifest.Images {
			artifact := &types.Artifact{Type: types.ArtifactImages, Name: img}
			if err := pkg.Get(artifact); err != nil {
				return err
			}
			fmt.Println("    ", artifact.Name, "\t", byteCountSI(artifact.Size))
			if inspectDetails {
				fmt.Println()
				imageNames, err := imageNamesFromTar(artifact.Body)
				if err != nil {
					fmt.Println("       - <", err.Error(), ">")
					continue
				}
				for _, i := range imageNames {
					fmt.Println("       -", i)
				}
				if i != len(meta.Manifest.Images)-1 {
					fmt.Println()
				}
			}
		}

		fmt.Println()
		fmt.Println("  MANIFESTS")
		for _, mani := range meta.Manifest.K8sManifests {
			artifact := &types.Artifact{Type: types.ArtifactManifest, Name: mani}
			if err := pkg.Get(artifact); err != nil {
				return err
			}
			fmt.Println("    ", artifact.Name, "\t", byteCountSI(artifact.Size))
		}

		fmt.Println()
		fmt.Println("  STATIC ASSETS")
		for _, static := range meta.Manifest.Static {
			artifact := &types.Artifact{Type: types.ArtifactStatic, Name: static}
			if err := pkg.Get(artifact); err != nil {
				return err
			}
			fmt.Println("    ", artifact.Name, "\t", byteCountSI(artifact.Size))
		}

		if cfg := meta.GetPackageConfig(); cfg != nil && len(cfg.Variables) > 0 {
			fmt.Println()
			fmt.Println("  PARAMETERS")
			for _, vari := range cfg.Variables {
				if vari.Default == "" {
					fmt.Println("    ", vari.Name, "(required)")
				} else {
					fmt.Println("    ", vari.Name, fmt.Sprintf("(default %q)", vari.Default))
				}
			}
			if inspectDetails {
				fmt.Println()
				fmt.Println("  CONFIG")
				scanner := bufio.NewScanner(bytes.NewReader(cfg.Raw))
				for scanner.Scan() {
					fmt.Println("    ", scanner.Text())
				}
			}
		}

		fmt.Println()
		return nil
	},
}

type imageManifest struct {
	RepoTags []string
	// not interested in anything else for now
}

func imageNamesFromTar(body io.ReadCloser) ([]string, error) {
	defer body.Close()
	out := make([]string, 0)
	reader := tar.NewReader(body)
	for {
		header, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				return nil, errors.New("no manifest.json found in the tar archive")
			}
			return nil, err
		}
		if header.Typeflag != tar.TypeReg || !strings.HasSuffix(header.Name, "manifest.json") {
			continue
		}
		manifestRaw, err := ioutil.ReadAll(reader)
		var imgs []imageManifest
		if err := json.Unmarshal(manifestRaw, &imgs); err != nil {
			return nil, err
		}
		for _, img := range imgs {
			out = append(out, img.RepoTags...)
		}
		return out, nil
	}
}

func byteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
