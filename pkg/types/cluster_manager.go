package types

// AddNodeOptions represents options passed to the AddNode operation.
type AddNodeOptions struct {
	// The user to attempt to SSH into the remote node as.
	SSHUser string
	// A password to use for SSH authentication.
	SSHPassword string
	// The path to the key to use for SSH authentication.
	SSHKeyFile string
	// The port to use for the SSH connection
	SSHPort int
	// The role to assign the new node.
	NodeRole K3sRole
	// The address of the new node.
	NodeAddress string
}

// ClusterManager is an interface for managing the nodes in a k3s cluster.
type ClusterManager interface {
	// AddNode should add a new node to the k3s cluster. It should only be
	// possible to use this method from the initial master instance.
	AddNode(*AddNodeOptions) error
}
