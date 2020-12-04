package types

// AddNodeOptions represents options passed to the AddNode operation.
type AddNodeOptions struct {
	*NodeConnectOptions
	// The role to assign the new node.
	NodeRole K3sRole
}

// ClusterManager is an interface for managing the nodes in a k3s cluster.
type ClusterManager interface {
	// AddNode should add a new node to the k3s cluster. It should only be
	// possible to use this method from the initial master instance.
	AddNode(*AddNodeOptions) error
}
