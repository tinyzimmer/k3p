package types

// NodeConnectOptions are options for configuring a connection to a remote node.
type NodeConnectOptions struct {
	// The user to attempt to SSH into the remote node as.
	SSHUser string
	// A password to use for SSH authentication.
	SSHPassword string
	// The path to the key to use for SSH authentication.
	SSHKeyFile string
	// The port to use for the SSH connection
	SSHPort int
	// The address of the new node.
	Address string
}

// AddNodeOptions represents options passed to the AddNode operation.
type AddNodeOptions struct {
	*NodeConnectOptions
	// The role to assign the new node.
	NodeRole K3sRole
}

// RemoveNodeOptions are options passed to a RemoveNode operation (not implemented).
type RemoveNodeOptions struct {
	*NodeConnectOptions
	Uninstall bool
	Name      string
	IPAddess  string
}

// ClusterManager is an interface for managing the nodes in a k3s cluster.
type ClusterManager interface {
	// AddNode should add a new node to the k3s cluster.
	AddNode(Node, *AddNodeOptions) error
	// RemoveNode should drain and remove the given node from the k3s cluster.
	// If NodeConnectOptions are not nil and Uninstall is true, then k3s and
	// all of its assets should be completely removed from the system. (not implemented)
	RemoveNode(*RemoveNodeOptions) error
}
