package types

import (
	"fmt"
	"strconv"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

const rancherRepo = "rancher/k3s"

// DockerNodeOptions are options for configuring a docker container
// as a k3s node.
type DockerNodeOptions struct {
	// The cluster options associated with this node
	ClusterOptions *DockerClusterOptions
	// The index for this node in the cluster
	NodeIndex int
	// The role of the node
	NodeRole K3sRole
}

// DockerClusterOptions represent options for building a cluster backed by docker
// containers.
type DockerClusterOptions struct {
	// The name of the cluster
	ClusterName string
	// The version of the k3s image to pull for this node
	K3sVersion string
	// The number of servers and agents to run in the cluster
	Servers, Agents int
	// The port on the host to bind the API to
	APIPort int
	// Additional port mappings to apply to the leader node
	PortMappings []string
}

// DockerClusterFilters returns the filters for matching all components of a given cluster.
func DockerClusterFilters(name string) filters.Args {
	return filters.NewArgs(filters.Arg("label", fmt.Sprintf("%s=%s", K3pDockerClusterLabel, name)))
}

// DockerOptionsFromContainer converts a container spec to docker node options.
func DockerOptionsFromContainer(container dockertypes.Container) *DockerNodeOptions {
	opts := &DockerNodeOptions{ClusterOptions: &DockerClusterOptions{}}

	opts.ClusterOptions.ClusterName = container.Labels[K3pDockerClusterLabel]
	opts.NodeRole = K3sRole(container.Labels[K3pDockerNodeRoleLabel])

	versFields := strings.Split(container.Image, ":")
	opts.ClusterOptions.K3sVersion = versFields[len(versFields)-1]

	nameFields := strings.Split(container.Names[0], "-")
	idx, err := strconv.Atoi(nameFields[len(nameFields)-1])
	if err == nil {
		opts.NodeIndex = idx
	}

	return opts
}

// GetLabels returns the labels for the node represented by these options.
func (d *DockerClusterOptions) GetLabels() map[string]string {
	return map[string]string{
		K3pDockerClusterLabel: d.ClusterName,
	}
}

// GetK3sImage returns the K3s image for these options
func (d *DockerNodeOptions) GetK3sImage() string {
	if d.NodeRole == K3sRoleLoadBalancer {
		return "rancher/k3d-proxy:latest"
	}
	return fmt.Sprintf("%s:%s", rancherRepo, strings.Replace(d.ClusterOptions.K3sVersion, "+", "-", -1))
}

// GetNodeName returns the name for the node represented by these options.
func (d *DockerNodeOptions) GetNodeName() string {
	nodeRole := K3sRoleServer
	if d.NodeRole != "" {
		nodeRole = d.NodeRole
	}
	if nodeRole == K3sRoleLoadBalancer {
		return fmt.Sprintf("%s-serverlb", strings.Replace(d.ClusterOptions.ClusterName, "_", "-", -1))
	}
	return fmt.Sprintf(
		"%s-%s-%s",
		strings.Replace(d.ClusterOptions.ClusterName, "_", "-", -1),
		string(nodeRole),
		strconv.Itoa(d.NodeIndex),
	)
}

// GetLabels returns the labels for the node represented by these options.
func (d *DockerNodeOptions) GetLabels() map[string]string {
	labels := d.ClusterOptions.GetLabels()
	labels[K3pDockerNodeNameLabel] = d.GetNodeName()
	nodeRole := string(K3sRoleServer)
	if d.NodeRole != "" {
		nodeRole = string(d.NodeRole)
	}
	labels[K3pDockerNodeRoleLabel] = nodeRole
	return labels
}

// GetComponentLabels returns the labels for a specific component represented by these options.
func (d *DockerNodeOptions) GetComponentLabels(component string) map[string]string {
	labels := d.GetLabels()
	labels["component"] = component
	return labels
}

// GetFilters returns the docker filters for the nodes represented by these options.
func (d *DockerNodeOptions) GetFilters() filters.Args {
	args := []filters.KeyValuePair{}
	for k, v := range d.GetLabels() {
		args = append(args, filters.Arg("label", fmt.Sprintf("%s=%s", k, v)))
	}
	return filters.NewArgs(args...)
}
