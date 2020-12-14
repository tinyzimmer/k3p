package cmd

import (
	"fmt"
	"net"
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
	nodeAddRole      string
	nodeRemoteLeader string
	nodeConnectOpts  *types.NodeConnectOptions
	nodeAddOpts      *types.AddNodeOptions
	nodeRemoveOpts   *types.RemoveNodeOptions
)

func init() {
	var currentUser *user.User
	var err error
	if currentUser, err = user.Current(); err != nil {
		log.Fatal(err)
	}

	nodeConnectOpts = &types.NodeConnectOptions{}
	nodeAddOpts = &types.AddNodeOptions{}
	nodeRemoveOpts = &types.RemoveNodeOptions{}

	var defaultKeyArg string
	defaultKeyPath := path.Join(currentUser.HomeDir, ".ssh", "id_rsa")
	if _, err := os.Stat(defaultKeyPath); err == nil {
		defaultKeyArg = defaultKeyPath
	}

	nodesCmd.PersistentFlags().StringVarP(&nodeConnectOpts.SSHUser, "ssh-user", "u", currentUser.Username, "The remote user to use for SSH authentication")
	nodesCmd.PersistentFlags().StringVarP(&nodeConnectOpts.SSHKeyFile, "private-key", "k", defaultKeyArg, "A private key to use for SSH authentication, if not provided you will be prompted for a password")
	nodesCmd.PersistentFlags().IntVarP(&nodeConnectOpts.SSHPort, "ssh-port", "p", 22, "The port to use when connecting to the remote instance over SSH")
	nodesCmd.PersistentFlags().StringVarP(&nodeRemoteLeader, "leader", "L", "", `The IP address or DNS name of the leader of the cluster.

When left unset, the machine running k3p is assumed to be the leader of the cluster. Otherwise,
the provided host is remoted into, with the same connection options as for the new node in case 
of an add, to retrieve the installation manifest.
`)

	nodesAddCmd.Flags().StringVarP(&nodeAddRole, "node-role", "r", string(types.K3sRoleAgent), "Whether to join the instance as a 'server' or 'agent'")
	nodesAddCmd.RegisterFlagCompletionFunc("node-role", completeStringOpts([]string{"server", "agent"}))

	nodesRemoveCmd.Flags().BoolVar(&nodeRemoveOpts.Uninstall, "uninstall", false, "After the node is removed from the cluster, remote in and uninstall k3s")

	nodesCmd.AddCommand(nodesAddCmd)
	nodesCmd.AddCommand(nodesRemoveCmd)

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
	RunE:  addNode,
}

var nodesRemoveCmd = &cobra.Command{
	Use:   "remove NODE [flags]",
	Short: "Remove a node from the cluster by name or IP",
	Args:  cobra.ExactArgs(1),
	RunE:  removeNode,
}

func addNode(cmd *cobra.Command, args []string) error {
	nodeAddOpts.NodeConnectOptions = nodeConnectOpts
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

	leader, err := getLeader(nodeAddOpts.Address)
	if err != nil {
		return err
	}

	log.Infof("Connecting to %s:%d\n", nodeAddOpts.Address, nodeAddOpts.SSHPort)
	newNode, err := node.Connect(nodeAddOpts.NodeConnectOptions)
	if err != nil {
		return err
	}

	return cluster.New(leader).AddNode(newNode, nodeAddOpts)
}

func removeNode(cmd *cobra.Command, args []string) error {
	if nodeRemoteLeader != "" && nodeConnectOpts.SSHKeyFile == "" {
		fmt.Printf("Enter SSH Password for %s: ", nodeAddOpts.SSHUser)
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}
		nodeConnectOpts.SSHPassword = string(bytePassword)
	}
	nodeRemoveOpts.NodeConnectOptions = nodeConnectOpts

	target := args[0]
	if ip := net.ParseIP(target); ip != nil {
		// Valid IP address
		nodeRemoveOpts.IPAddress = target
	} else {
		// Assume it's a node name
		nodeRemoveOpts.Name = target
	}

	leader, err := getLeader(target)
	if err != nil {
		return err
	}

	return cluster.New(leader).RemoveNode(nodeRemoveOpts)
}

func getLeader(nodeName string) (types.Node, error) {
	if nodeRemoteLeader != "" {
		connectOpts := *nodeConnectOpts
		connectOpts.Address = nodeRemoteLeader
		log.Infof("Connecting to %s:%d\n", connectOpts.Address, connectOpts.SSHPort)
		return node.Connect(&connectOpts)
	}
	return node.Local(), nil
}
