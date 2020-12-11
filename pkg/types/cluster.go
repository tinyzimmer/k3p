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
	// Options for remote connections
	*NodeConnectOptions
	// The role to assign the new node.
	NodeRole K3sRole
}

// RemoveNodeOptions are options passed to a RemoveNode operation (not implemented).
type RemoveNodeOptions struct {
	// Options for remote connections
	*NodeConnectOptions
	// Attempt to remote into the system and uninstall k3s
	Uninstall bool
	// The name of the node to remove
	Name string
	// The IP address of the node to remove
	IPAddress string
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
