package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	v1 "github.com/tinyzimmer/k3p/pkg/build/package/v1"
	"github.com/tinyzimmer/k3p/pkg/types"
)

func init() {
	rootCmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect PACKAGE",
	Short: "Inspect the given package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		pkg, err := v1.Load(f)
		if err != nil {
			return err
		}
		defer pkg.Close()

		meta := pkg.GetMeta()

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
		fmt.Println("  IMAGES")
		for _, img := range meta.Manifest.Images {
			artifact := &types.Artifact{Type: types.ArtifactImages, Name: img}
			if err := pkg.Get(artifact); err != nil {
				return err
			}
			fmt.Println("    ", artifact.Name, "\t", byteCountSI(artifact.Size))
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
		return nil
	},
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
