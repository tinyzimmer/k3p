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
