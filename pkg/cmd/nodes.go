package cmd

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/tinyzimmer/k3p/pkg/cluster"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var (
	nodeAddRole string
	nodeAddOpts *types.AddNodeOptions
)

func init() {
	nodeAddOpts = &types.AddNodeOptions{NodeConnectOptions: &types.NodeConnectOptions{}}

	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	var defaultKeyArg string
	defaultKeyPath := path.Join(u.HomeDir, ".ssh", "id_rsa")
	if _, err := os.Stat(defaultKeyPath); err == nil {
		defaultKeyArg = defaultKeyPath
	}

	nodesAddCmd.Flags().StringVarP(&nodeAddOpts.SSHUser, "ssh-user", "u", u.Username, "The remote user to use for SSH authentication")
	nodesAddCmd.Flags().StringVarP(&nodeAddOpts.SSHKeyFile, "private-key", "k", defaultKeyArg, "A private key to use for SSH authentication, if not provided you will be prompted for a password")
	nodesAddCmd.Flags().IntVarP(&nodeAddOpts.SSHPort, "ssh-port", "p", 22, "The port to use when connecting to the remote instance over SSH")
	nodesAddCmd.Flags().StringVarP(&nodeAddRole, "node-role", "r", string(types.K3sRoleAgent), "Whether to join the instance as a 'server' or 'agent'")

	nodesCmd.AddCommand(nodesAddCmd)
	rootCmd.AddCommand(nodesCmd)
}

var nodesCmd = &cobra.Command{
	Use:   "node",
	Short: "Node management commands",
}

var nodesAddCmd = &cobra.Command{
	Use:   "add NODE [flags]",
	Short: "Add a new node to the cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeAddOpts.Address = args[0]

		switch types.K3sRole(nodeAddRole) {
		case types.K3sRoleServer:
			nodeAddOpts.NodeRole = types.K3sRoleServer
		case types.K3sRoleAgent:
			nodeAddOpts.NodeRole = types.K3sRoleAgent
		default:
			return fmt.Errorf("%q is not a valid node role", nodeRole)
		}

		if nodeAddOpts.SSHKeyFile == "" {
			fmt.Printf("Enter SSH Password for %s: ", nodeAddOpts.SSHUser)
			bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return err
			}
			nodeAddOpts.SSHPassword = string(bytePassword)
		}

		return cluster.New().AddNode(nodeAddOpts)
	},
}
