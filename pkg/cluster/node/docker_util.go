package node

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

// LoadDockerCluster will load all the nodes associated with a docker cluster. Close only needs
// to be run on one of the returned nodes, as they all share an underlying client.
func LoadDockerCluster(name string) ([]*Docker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	containers, err := cli.ContainerList(context.TODO(), dockertypes.ContainerListOptions{
		Filters: types.DockerClusterFilters(name),
	})
	if err != nil {
		return nil, err
	}
	nodes := make([]*Docker, len(containers))
	for i, container := range containers {
		nodes[i] = &Docker{
			cli:         cli,
			containerID: container.ID,
			opts:        types.DockerOptionsFromContainer(container),
		}
	}
	return nodes, nil
}

// DeleteDockerNetwork deletes a docker network with the given name
func DeleteDockerNetwork(name string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	return cli.NetworkRemove(context.TODO(), name)
}

func pullIfNotPresent(cli client.APIClient, image string) error {
	imgs, err := cli.ImageList(context.TODO(), dockertypes.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", image)),
	})
	if err != nil {
		return err
	}
	if len(imgs) == 1 {
		log.Debugf("Image %s already exists on the system\n", image)
		return nil
	}
	log.Infof("Pulling image for %s\n", image)
	rdr, err := cli.ImagePull(context.TODO(), image, dockertypes.ImagePullOptions{})
	if err != nil {
		return err
	}
	log.DebugReader(rdr)
	return nil
}

func ensureClusterNetwork(cli client.APIClient, opts *types.DockerClusterOptions) error {
	networks, err := cli.NetworkList(context.TODO(), dockertypes.NetworkListOptions{Filters: types.DockerClusterFilters(opts.ClusterName)})
	if err != nil {
		return err
	}
	if len(networks) > 0 {
		return nil
	}
	log.Info("Creating docker network", opts.ClusterName)
	_, err = cli.NetworkCreate(context.TODO(), opts.ClusterName, dockertypes.NetworkCreate{
		Labels:         opts.GetLabels(),
		CheckDuplicate: true,
	})
	return err
}

func buildDockerEnv(nodeOpts *types.DockerNodeOptions, opts *types.ExecuteOptions) strslice.StrSlice {
	out := make([]string, 0)
	if nodeOpts.NodeRole == types.K3sRoleLoadBalancer {
		servers := make([]string, 0)
		for i := 0; i < nodeOpts.ClusterOptions.Servers; i++ {
			serverOpts := &types.DockerNodeOptions{NodeRole: types.K3sRoleServer, NodeIndex: i, ClusterOptions: nodeOpts.ClusterOptions}
			servers = append(servers, serverOpts.GetNodeName())
		}
		out = append(out, fmt.Sprintf("SERVERS=%s", strings.Join(servers, ",")))
		ports := []string{"6443"}
		for _, port := range nodeOpts.ClusterOptions.PortMappings {
			spl := strings.Split(port, "@")
			if len(spl) == 1 || spl[len(spl)-1] == string(types.K3sRoleLoadBalancer) {
				portSpl := strings.Split(spl[0], ":")
				ports = append(ports, portSpl[len(portSpl)-1])
			}
		}
		out = append(out, fmt.Sprintf("PORTS=%s", strings.Join(ports, ",")))
		return strslice.StrSlice(out)
	}
	for k, v := range opts.Env {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return strslice.StrSlice(out)
}

func buildDockerCmd(nodeOpts *types.DockerNodeOptions, opts *types.ExecuteOptions) strslice.StrSlice {
	if nodeOpts.NodeRole == types.K3sRoleLoadBalancer {
		return nil
	}
	fields := strings.Fields(opts.Command)
	var cmd []string
	switch fields[len(fields)-1] {
	case string(types.K3sRoleAgent):
		cmd = []string{string(types.K3sRoleAgent)}
	default:
		cmd = []string{string(types.K3sRoleServer), "--tls-san", "0.0.0.0"}
	}
	for k, v := range opts.Env {
		if k == "INSTALL_K3S_EXEC" {
			cmd = append(cmd, strings.Fields(v)...)
		}
	}
	return strslice.StrSlice(cmd)
}

var portMapRegex = regexp.MustCompile("(?P<Role>[a-z]+)(?P<Slice>\\[(?P<Index>[0-9])\\])?")

func parsePortMapping(opts *types.DockerNodeOptions, portMapping string) (exposedPorts map[nat.Port]struct{}, portBindings map[nat.Port][]nat.PortBinding) {
	spl := strings.Split(portMapping, "@")
	if len(spl) == 1 {
		// Default to loadbalancer only
		if opts.NodeRole != types.K3sRoleLoadBalancer {
			return nil, nil
		}
		exposedPorts, portBindings, err := nat.ParsePortSpecs([]string{portMapping})
		if err != nil {
			log.Errorf("Error parsing %q, ignoring: %s\n", portMapping, err.Error())
			return nil, nil
		}
		return exposedPorts, portBindings
	}

	matches := portMapRegex.FindStringSubmatch(spl[1])
	if opts.NodeRole != types.K3sRole(matches[1]) {
		log.Debugf("Port mapping %q does not match current node role %s\n", portMapping, opts.NodeRole)
		return nil, nil
	}
	if opts.NodeRole == types.K3sRoleLoadBalancer {
		exposedPorts, portBindings, err := nat.ParsePortSpecs([]string{spl[0]})
		if err != nil {
			log.Errorf("Error parsing %q, ignoring: %s\n", portMapping, err.Error())
			return nil, nil
		}
		return exposedPorts, portBindings
	}
	// If server or agent check the index
	if matches[3] == "" {
		log.Errorf("Ignoring %q: Servers and agents must have an index (e.g. server[0])\n", portMapping)
		return nil, nil
	}

	index, err := strconv.Atoi(matches[3])
	if err != nil {
		log.Errorf("Invalid integer for node index %q ignoring: %s\n", matches[3], err.Error())
		return nil, nil
	}

	if index != opts.NodeIndex {
		log.Debugf("Port mapping %s does not match the current node index %d\n", portMapping, opts.NodeIndex)
		return nil, nil
	}

	exposedPorts, portBindings, err = nat.ParsePortSpecs([]string{spl[0]})
	if err != nil {
		log.Errorf("Error parsing %q, ignoring: %s\n", portMapping, err.Error())
		return nil, nil
	}

	return exposedPorts, portBindings
}

func translateOptsToConfigs(opts *types.DockerNodeOptions, execOpts *types.ExecuteOptions) (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "always"},
		PortBindings:  map[nat.Port][]nat.PortBinding{},
	}
	if opts.NodeRole != types.K3sRoleLoadBalancer {
		hostConfig.Privileged = true
		hostConfig.SecurityOpt = []string{"label=disable"}
		hostConfig.Init = &[]bool{true}[0]
		hostConfig.Binds = []string{fmt.Sprintf("%s:%s", opts.GetNodeName(), types.K3sRootConfigDir)}
		hostConfig.Tmpfs = map[string]string{
			"/run":     "",
			"/var/run": "",
		}
	}
	containerConfig := &container.Config{
		Hostname:     opts.GetNodeName(),
		Image:        opts.GetK3sImage(),
		Env:          buildDockerEnv(opts, execOpts),
		Cmd:          buildDockerCmd(opts, execOpts),
		Labels:       opts.GetComponentLabels(fmt.Sprintf("k3s-%s", string(opts.NodeRole))),
		ExposedPorts: map[nat.Port]struct{}{},
	}
	if opts.NodeRole != types.K3sRoleLoadBalancer {
		containerConfig.Volumes = map[string]struct{}{
			types.K3sRootConfigDir: struct{}{},
		}
	}
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			opts.ClusterOptions.ClusterName: &network.EndpointSettings{
				Aliases: []string{opts.GetNodeName()},
			},
		},
	}
	ports := append(opts.ClusterOptions.PortMappings, fmt.Sprintf("%d:6443/tcp@loadbalancer", opts.ClusterOptions.APIPort))
	for _, port := range ports {
		exposedPorts, portBindings := parsePortMapping(opts, port)
		for k, v := range exposedPorts {
			containerConfig.ExposedPorts[k] = v
		}
		for k, v := range portBindings {
			hostConfig.PortBindings[k] = v
		}
	}

	return containerConfig, hostConfig, networkConfig, nil
}
