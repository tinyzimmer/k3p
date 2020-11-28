package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	v1 "github.com/tinyzimmer/k3p/pkg/archive/v1"
)

func init() {
	rootCmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect PACKAGE",
	Short: "Inspect the given package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pkg, err := v1.Load(args[0], tmpDir)
		if err != nil {
			return err
		}
		defer pkg.Close()
		manifest, err := pkg.GetManifest()
		if err != nil {
			return err
		}
		fmt.Println("NAME:", args[0])
		fmt.Println()
		fmt.Println("BINARIES")
		for _, bin := range manifest.Bins {
			fmt.Println("    ", bin.Name, "\t", byteCountSI(bin.Size))
		}
		fmt.Println()
		fmt.Println("SCRIPTS")
		for _, sc := range manifest.Scripts {
			fmt.Println("    ", sc.Name, "\t", byteCountSI(sc.Size))
		}
		fmt.Println()
		fmt.Println("IMAGES")
		for _, img := range manifest.Images {
			fmt.Println("    ", img.Name, "\t", byteCountSI(img.Size))
		}
		fmt.Println()
		fmt.Println("MANIFESTS")
		for _, mani := range manifest.Manifests {
			fmt.Println("    ", mani.Name, "\t", byteCountSI(mani.Size))
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
