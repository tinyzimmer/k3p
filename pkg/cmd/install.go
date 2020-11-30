package cmd

import (
	"errors"
	"os/user"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/install"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var (
	installAcceptEULA     bool
	installNodeName       string
	installJoinHost       string
	installJoinToken      string
	installResolvConf     string
	installKubeconfigMode string
	installK3sExecArgs    string
)

func init() {
	installCmd.Flags().StringVarP(&installNodeName, "node-name", "n", "", "An optional name to give this node in the cluster")
	installCmd.Flags().BoolVar(&installAcceptEULA, "accept-eula", false, "Automatically accept any EULA included with the package")
	installCmd.Flags().StringVarP(&installJoinHost, "join", "j", "", "When installing an agent instance, the address of the server to join (e.g. https://myserver:6443)")
	installCmd.Flags().StringVarP(&installJoinToken, "token", "t", "", "When installing an agent instance, the node token from the server (typically found at /var/lib/rancher/k3s/server/node-token)")
	installCmd.Flags().StringVar(&installResolvConf, "resolv-conf", "", "The path of a resolv-conf file to use when configuring DNS in the cluster")
	installCmd.Flags().StringVar(&installKubeconfigMode, "kubeconfig-mode", "", "The mode to set on the k3s kubeconfig. Default is to only allow root access")
	installCmd.Flags().StringVar(&installK3sExecArgs, "k3s-exec", "", "Extra arguments to pass to the k3s server or agent process")

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
		err = install.New().Install(&types.InstallOptions{
			TarPath:        args[0],
			NodeName:       installNodeName,
			AcceptEULA:     installAcceptEULA,
			ServerURL:      installJoinHost,
			NodeToken:      installJoinToken,
			ResolvConf:     installResolvConf,
			KubeconfigMode: installKubeconfigMode,
			K3sExecArgs:    installK3sExecArgs,
		})
		if err != nil {
			return err
		}

		log.Info("The cluster has been installed. For additional details run `kubectl cluster-info`.")

		return nil
	},
}
