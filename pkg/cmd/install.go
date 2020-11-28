package cmd

import (
	"errors"
	"os/user"

	"github.com/spf13/cobra"

	v1 "github.com/tinyzimmer/k3p/pkg/archive/v1"
	"github.com/tinyzimmer/k3p/pkg/install"
)

func init() {
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install PACKAGE",
	Short: "Install the given package to the system (requires root)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		usr, err := user.Current()
		if err != nil {
			return err
		}
		if usr.Uid != "0" {
			return errors.New("Install must be run as root")
		}
		pkg, err := v1.Load(args[0], tmpDir)
		if err != nil {
			return err
		}
		defer pkg.Close()
		manifest, err := pkg.GetManifest()
		if err != nil {
			return err
		}
		return install.New().Install(manifest, &install.Options{})
	},
}
