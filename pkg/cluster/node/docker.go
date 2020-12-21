package node

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// NewDocker initializes a new node using a local container for the instance.
func NewDocker(opts *types.DockerNodeOptions) (types.Node, error) {
	// Get a docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	// Check if the node exists already, if it does everything else is probably done
	containers, err := cli.ContainerList(context.TODO(), dockertypes.ContainerListOptions{
		Filters: opts.GetFilters(),
	})
	if err != nil {
		return nil, err
	}
	// If we have a container, return it
	if len(containers) == 1 {
		return &Docker{
			cli:         cli,
			containerID: containers[0].ID,
			opts:        opts,
		}, nil
	}

	// Ensure a docker network for the cluster
	if err := ensureClusterNetwork(cli, opts.ClusterOptions); err != nil {
		return nil, err
	}

	// Ensure the docker image with the given k3s version is ready for when we start
	if err := pullIfNotPresent(cli, opts.GetK3sImage()); err != nil {
		return nil, err
	}

	// If a load balancer, immediately return a non-initialized container for Execute
	// to create/start
	if opts.NodeRole == types.K3sRoleLoadBalancer {
		return &Docker{cli: cli, opts: opts}, nil
	}

	// Create a volume for k3s assets
	varVolCreateBody := volume.VolumeCreateBody{
		Driver:     "local",
		DriverOpts: map[string]string{},
		Labels:     opts.GetLabels(),
		Name:       opts.GetNodeName(),
	}
	etcVolCreateBody := volume.VolumeCreateBody{
		Driver:     "local",
		DriverOpts: map[string]string{},
		Labels:     opts.GetLabels(),
		Name:       fmt.Sprintf("%s-etc", opts.GetNodeName()),
	}
	volNames := make([]string, 2)
	for i, volCreateBody := range []volume.VolumeCreateBody{varVolCreateBody, etcVolCreateBody} {
		log.Info("Creating docker volume", volCreateBody.Name)
		log.Debugf("VolumeCreateBody: %+v\n", volCreateBody)
		vol, err := cli.VolumeCreate(context.TODO(), volCreateBody)
		if err != nil {
			return nil, err
		}
		volNames[i] = vol.Name
	}

	// We at first just use a busybox container with a persistent volume to serve
	// GetFile and WriteFile requests. K3s is not actually launched until Execute
	// is called.
	if err := pullIfNotPresent(cli, "busybox:latest"); err != nil {
		return nil, err
	}
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:%s", volNames[0], types.K3sRootConfigDir),
			fmt.Sprintf("%s:%s", volNames[1], types.K3sEtcDir),
		},
	}
	busyboxConfig := &container.Config{
		Image: "busybox:latest",
		Volumes: map[string]struct{}{
			types.K3sRootConfigDir: struct{}{},
			types.K3sEtcDir:        struct{}{},
		},
		Labels: opts.GetComponentLabels("busybox"),
		Cmd:    strslice.StrSlice([]string{"tail", "-f", "/dev/null"}),
	}
	log.Debugf("Busybox container config: %+v\n", busyboxConfig)
	log.Debugf("Busybox host config: %+v\n", hostConfig)
	log.Debug("Creating busybox container")
	container, err := cli.ContainerCreate(context.TODO(), busyboxConfig, hostConfig, nil, opts.GetNodeName())
	if err != nil {
		return nil, err
	}
	log.Debugf("Starting busybox container %q\n", container.ID)
	if err := cli.ContainerStart(context.TODO(), container.ID, dockertypes.ContainerStartOptions{}); err != nil {
		return nil, err
	}

	return &Docker{
		cli:         cli,
		containerID: container.ID,
		opts:        opts,
	}, nil
}

// Docker represents a node backed by a docker container. It is exported for the extra
// methods it provides.
type Docker struct {
	cli         client.APIClient
	containerID string
	opts        *types.DockerNodeOptions
}

// GetType implements the node interface.
func (d *Docker) GetType() types.NodeType { return types.NodeDocker }

// MkdirAll implements the node interface and will create a directory inside the current
// container.
func (d *Docker) MkdirAll(dir string) error {
	execCfg := dockertypes.ExecConfig{
		User:   "root",
		Cmd:    []string{"mkdir", "-p", dir},
		Detach: true,
	}
	log.Debugf("Creating exec process in container %q: %+v\n", d.containerID, execCfg)
	id, err := d.cli.ContainerExecCreate(context.TODO(), d.containerID, execCfg)
	if err != nil {
		return err
	}
	if err := d.cli.ContainerExecStart(context.TODO(), id.ID, dockertypes.ExecStartCheck{}); err != nil {
		return err
	}
	for {
		status, err := d.cli.ContainerExecInspect(context.TODO(), id.ID)
		if err != nil {
			return err
		}
		if status.Pid == 0 { // process hasn't started yet
			continue
		}
		if status.Running { // process is still running
			continue
		}
		if status.ExitCode == 0 { // process completed
			return nil
		}
		// process exited with error
		return fmt.Errorf("process exited with status code %d", status.ExitCode)
	}
}

// GetFile implements the node interface and will retrieve a file from the container.
func (d *Docker) GetFile(path string) (io.ReadCloser, error) {
	rdr, _, err := d.cli.CopyFromContainer(context.TODO(), d.containerID, path)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(rdr)
	header, err := tr.Next()
	if err != nil {
		return nil, err
	}
	if header.Typeflag != tar.TypeReg {
		if header.Typeflag == tar.TypeSymlink {
			log.Debugf("Following symlink to %q\n", header.Linkname)
			return d.GetFile(header.Linkname)
		}
		log.Debugf("Invalid header: %+v\n", *header)
		return nil, fmt.Errorf("%q is not a regular file", path)
	}
	return ioutil.NopCloser(tr), nil
}

// WriteFile implements the node interface, and will write a file to the container. For docker nodes
// it only accepts files rooted in /var/lib/rancher/k3s.
func (d *Docker) WriteFile(rdr io.ReadCloser, destination string, mode string, size int64) error {
	defer rdr.Close()

	// stupid hack to only care about actual runtime files
	if !strings.HasPrefix(destination, types.K3sRootConfigDir) && !strings.HasPrefix(destination, types.K3sEtcDir) {
		return nil
	}
	if err := d.MkdirAll(path.Dir(destination)); err != nil {
		return err
	}

	// Make a pipe for sending the contents to the container
	r, w := io.Pipe()

	// Kick off the copy in a goroutine
	errors := make(chan error)
	log.Debugf("Spawning copy process to %q in container %q\n", path.Dir(destination), d.containerID)
	go func() {
		errors <- d.cli.CopyToContainer(context.TODO(), d.containerID, path.Dir(destination), r, dockertypes.CopyToContainerOptions{})
	}()

	// Write tar data to the pipe
	tw := tar.NewWriter(w)

	modeInt, err := strconv.ParseInt(mode, 0, 16)
	if err != nil {
		return err
	}
	now := time.Now()
	header := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     path.Base(destination),
		Size:     size,
		Mode:     modeInt,
		Uid:      0, Gid: 0,
		Uname: "root", Gname: "root",
		ModTime: now, AccessTime: now, ChangeTime: now,
	}
	log.Debugf("Generated tar header for docker copy: %+v\n", header)

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	log.Debugf("Copying tar buffer to container %q at %q\n", d.containerID, destination)
	if _, err := io.Copy(tw, rdr); err != nil {
		log.Error("Error copying contents to buffer:", err)
	}

	if err := tw.Close(); err != nil {
		return err
	}

	// Send an EOF to the docker copy
	if err := w.Close(); err != nil {
		return err
	}

	return <-errors
}

// Execute implements the node interface and starts the K3s container. It treats the provided command
// as arguments to run K3s with. It's because of this implementation that this method should probably be
// renamed.
func (d *Docker) Execute(opts *types.ExecuteOptions) error {
	if opts.Command == "k3s-uninstall.sh" { // TODO: type cast this somewhere
		return d.RemoveAll()
	}
	log.Info("Starting k3s docker node", d.opts.GetNodeName())
	// Assume being called to start K3s if a server or agent, first remove busybox container
	if d.opts.NodeRole != types.K3sRoleLoadBalancer {
		log.Debug("Removing busybox bootstrap node for", d.opts.GetNodeName())
		if err := d.cli.ContainerRemove(context.TODO(), d.containerID, dockertypes.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			return err
		}
	}
	// Build the k3s container according to the opts
	containerConfig, hostConfig, networkConfig, err := translateOptsToConfigs(d.opts, opts)
	if err != nil {
		return err
	}
	log.Debugf("K3s container config: %+v\n", containerConfig)
	log.Debugf("K3s host config: %+v\n", hostConfig)
	log.Debugf("K3s network config: %+v\n", networkConfig)
	container, err := d.cli.ContainerCreate(context.TODO(), containerConfig, hostConfig, networkConfig, d.opts.GetNodeName())
	if err != nil {
		return err
	}
	log.Debugf("Starting K3s container %q\n", container.ID)
	if err := d.cli.ContainerStart(context.TODO(), container.ID, dockertypes.ContainerStartOptions{}); err != nil {
		return err
	}
	d.containerID = container.ID
	return nil
}

// GetK3sAddress implements the node interface and returns this node's name. It is assumed
// the interested caller is interacting with a node on the same network.
func (d *Docker) GetK3sAddress() (string, error) {
	return d.opts.GetNodeName(), nil
}

// Close implements the node interface and closes the connection to the docker daemon
func (d *Docker) Close() error { return d.cli.Close() }

// RemoveAll is a special method implemented by the Docker object. It cleans up the container
// and all its resources.
func (d *Docker) RemoveAll() error {
	if addr, err := d.GetK3sAddress(); err == nil { // it's always nil for docker
		log.Info("Removing docker container and volumes for", addr)
	}
	if err := d.cli.ContainerRemove(context.TODO(), d.containerID, dockertypes.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		return err
	}
	for _, vol := range []string{d.opts.GetNodeName(), fmt.Sprintf("%s-etc", d.opts.GetNodeName())} {
		if err := d.cli.VolumeRemove(context.TODO(), vol, true); err != nil {
			return err
		}
	}
	return nil
}

// IsK3sRunning is a special method implemented by the Docker object to determine if a node is already
// running k3s.
func (d *Docker) IsK3sRunning() bool {
	status, err := d.cli.ContainerInspect(context.TODO(), d.containerID)
	if err != nil {
		// Assume CLI error is false, might not be a good idea tho
		return false
	}
	if len(status.Config.Entrypoint) == 0 {
		return false
	}
	return strings.HasSuffix(status.Config.Entrypoint[0], "k3s") && status.State.Running
}

// GetOpts returns the options that were used to configure this node.
func (d *Docker) GetOpts() *types.DockerNodeOptions { return d.opts }
