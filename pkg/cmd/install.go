package cmd

import (
	"errors"
	"fmt"
	"os/user"

	"github.com/spf13/cobra"

	"github.com/tinyzimmer/k3p/pkg/install"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var (
	nodeRole    string
	installOpts *types.InstallOptions
)

func init() {
	installOpts = &types.InstallOptions{}

	installCmd.Flags().StringVarP(&installOpts.NodeName, "node-name", "n", "", "An optional name to give this node in the cluster")
	installCmd.Flags().BoolVar(&installOpts.AcceptEULA, "accept-eula", false, "Automatically accept any EULA included with the package")
	installCmd.Flags().StringVarP(&installOpts.ServerURL, "join", "j", "", "When installing an agent instance, the address of the server to join (e.g. https://myserver:6443)")
	installCmd.Flags().StringVarP(&nodeRole, "join-role", "r", "agent", `Specify whether to join the cluster as a "server" or "agent"`)
	installCmd.Flags().StringVarP(&installOpts.NodeToken, "token", "t", "", `When installing an additional agent or server instance, the node token to use
(Found at "/var/lib/rancher/k3s/server/node-token" for new agents and generated from "--init-ha" for new servers)`)
	installCmd.Flags().StringVar(&installOpts.ResolvConf, "resolv-conf", "", "The path of a resolv-conf file to use when configuring DNS in the cluster")
	installCmd.Flags().StringVar(&installOpts.KubeconfigMode, "kubeconfig-mode", "", "The mode to set on the k3s kubeconfig. Default is to only allow root access")
	installCmd.Flags().StringVar(&installOpts.K3sExecArgs, "k3s-exec", "", "Extra arguments to pass to the k3s server or agent process")
	installCmd.Flags().BoolVar(&installOpts.InitHA, "init-ha", false, `When set, this server will run with the --cluster-init flag to enable clustering, 
and a token will be generated for adding additional servers to the cluster with "--join-role server"`)

	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install PACKAGE",
	Short: "Install the given package to the system (requires root)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// make sure we are root
		usr, err := user.Current()
		if err != nil {
			return err
		}
		if usr.Uid != "0" {
			return errors.New("Install must be run as root")
		}

		// Assign the package path to the opts
		installOpts.TarPath = args[0]

		// check the node role to make sure it's valid if relevant
		if installOpts.ServerURL != "" && nodeRole != "" {
			switch types.K3sRole(nodeRole) {
			case types.K3sRoleServer:
				installOpts.K3sRole = types.K3sRoleServer
			case types.K3sRoleAgent:
				installOpts.K3sRole = types.K3sRoleAgent
			default:
				return fmt.Errorf("%q is not a valid node role", nodeRole)
			}
		}

		// run the installation
		err = install.New().Install(installOpts)
		if err != nil {
			return err
		}

		log.Info("The cluster has been installed. For additional details run `kubectl cluster-info`.")
		return nil
	},
}
