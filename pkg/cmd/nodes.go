package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/tinyzimmer/k3p/pkg/cluster"
	"github.com/tinyzimmer/k3p/pkg/cluster/node"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var (
	nodeAddRole         string
	nodeAddRemoteLeader string
	nodeAddOpts         *types.AddNodeOptions
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
	nodesAddCmd.Flags().StringVarP(&nodeAddRemoteLeader, "leader", "L", "", `The IP address or DNS name of the leader of the cluster.

When left unset, the machine running k3p is assumed to be the leader of the cluster. Otherwise,
the provided host is remoted into, with the same connection options as for the new node, to retrieve
the installation manifest.
`)

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
			return fmt.Errorf("%q is not a valid node role", nodeAddRole)
		}

		if nodeAddOpts.SSHKeyFile == "" {
			fmt.Printf("Enter SSH Password for %s: ", nodeAddOpts.SSHUser)
			bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return err
			}
			nodeAddOpts.SSHPassword = string(bytePassword)
		}

		var leader types.Node
		var err error
		if nodeAddRemoteLeader != "" {
			connectOpts := *nodeAddOpts.NodeConnectOptions
			connectOpts.Address = nodeAddRemoteLeader
			log.Infof("Connecting to %s:%d\n", connectOpts.Address, connectOpts.SSHPort)
			leader, err = node.Connect(&connectOpts)
			if err != nil {
				return err
			}
		} else {
			leader = node.Local()
		}

		log.Infof("Connecting to %s:%d\n", nodeAddOpts.Address, nodeAddOpts.SSHPort)
		newNode, err := node.Connect(nodeAddOpts.NodeConnectOptions)
		if err != nil {
			return err
		}

		return cluster.New(leader).AddNode(newNode, nodeAddOpts)
	},
}
