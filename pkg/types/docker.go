package types

// DockerNodeOptions are options for configuring a docker container
// as a k3s node.
type DockerNodeOptions struct {
	Name       string
	K3sVersion string
}
