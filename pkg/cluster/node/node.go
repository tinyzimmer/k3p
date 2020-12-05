package node

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"

	"golang.org/x/crypto/ssh"
)

// SyncManifestToNode is a convenience method for extracting the contents of a package manifest
// to a k3s node.
func SyncManifestToNode(system types.Node, manifest *types.PackageManifest) error {
	log.Info("Installing binaries to remote machine at", types.K3sBinDir)
	for _, bin := range manifest.Bins {
		if err := system.WriteFile(bin.Body, path.Join(types.K3sBinDir, bin.Name), "0755", bin.Size); err != nil {
			return err
		}
	}

	log.Info("Installing scripts to remote machine at", types.K3sScriptsDir)
	for _, script := range manifest.Scripts {
		if err := system.WriteFile(script.Body, path.Join(types.K3sScriptsDir, script.Name), "0755", script.Size); err != nil {
			return err
		}
	}

	log.Info("Installing images to remote machine at", types.K3sImagesDir)
	for _, imgs := range manifest.Images {
		if err := system.WriteFile(imgs.Body, path.Join(types.K3sImagesDir, imgs.Name), "0644", imgs.Size); err != nil {
			return err
		}
	}

	log.Info("Installing manifests to remote machine at", types.K3sManifestsDir)
	for _, mani := range manifest.Manifests {
		if err := system.WriteFile(mani.Body, path.Join(types.K3sManifestsDir, mani.Name), "0644", mani.Size); err != nil {
			return err
		}
	}
	return nil
}

// Local returns a new Node pointing at the local system.
func Local() types.Node {
	return &localNode{}
}

type localNode struct{}

func (l *localNode) MkdirAll(dir string) error {
	log.Debugf("Ensuring local system directory %q with mode 0755\n", dir)
	return os.MkdirAll(dir, 0755)
}

func (l *localNode) Close() error { return nil }

// size is ignored for local nodes
func (l *localNode) WriteFile(rdr io.ReadCloser, dest string, mode string, size int64) error {
	log.Debugf("Writing file to local system at %q with mode %q\n", dest, mode)
	defer rdr.Close()
	if err := l.MkdirAll(path.Dir(dest)); err != nil {
		return err
	}
	u, err := strconv.ParseUint("0755", 0, 16)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, os.FileMode(u))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, rdr)
	return err
}

func (l *localNode) Execute(cmd string, logPrefix string) error {
	c := exec.Command("/bin/sh", "-c", cmd)
	outPipe, err := c.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := c.StderrPipe()
	if err != nil {
		return err
	}
	go log.TailReader(logPrefix, outPipe)
	go log.TailReader(logPrefix, errPipe)
	return c.Run()
}

// Connect will connect to a node over SSH with the given options.
func Connect(opts *types.NodeConnectOptions) (types.Node, error) {
	var err error
	n := &remoteNode{}
	n.client, err = getSSHClient(opts)
	return n, err
}

type remoteNode struct {
	client *ssh.Client
}

func (n *remoteNode) scpClient() (*scp.Client, error) {
	scpClient, err := scp.NewClientBySSH(n.client)
	if err != nil {
		return nil, err
	}
	scpClient.RemoteBinary = "sudo scp"
	return &scpClient, nil
}

func (n *remoteNode) WriteFile(rdr io.ReadCloser, destination string, mode string, size int64) error {
	if err := n.MkdirAll(path.Dir(destination)); err != nil {
		return err
	}
	scpClient, err := n.scpClient()
	if err != nil {
		return err
	}
	defer scpClient.Close()
	defer rdr.Close()
	log.Debugf("Sending %d bytes of %q to %q and setting mode to %s\n", size, path.Base(destination), destination, mode)
	return scpClient.Copy(rdr, destination, mode, size)
}

func (n *remoteNode) MkdirAll(dir string) error {
	sess, err := n.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	cmd := fmt.Sprintf("sudo mkdir -p %s", dir)
	log.Debug("Running command on remote:", cmd)
	return sess.Run(cmd)
}

func (n *remoteNode) Execute(cmd string, logPrefix string) error {
	sess, err := n.client.NewSession()
	if err != nil {
		return err
	}
	outPipe, err := sess.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := sess.StderrPipe()
	if err != nil {
		return err
	}
	go log.TailReader(logPrefix, outPipe)
	go log.TailReader(logPrefix, errPipe)
	return sess.Run(cmd)
}

func (n *remoteNode) Close() error { return n.client.Close() }

func newScpClient(client *ssh.Client) (scp.Client, error) {
	scpClient, err := scp.NewClientBySSH(client)
	if err != nil {
		return scp.Client{}, err
	}
	scpClient.RemoteBinary = "sudo scp"
	return scpClient, nil
}

func getSSHClient(opts *types.NodeConnectOptions) (*ssh.Client, error) {
	log.Debug("Using SSH user:", opts.SSHUser)
	config := &ssh.ClientConfig{
		User:            opts.SSHUser,
		Auth:            make([]ssh.AuthMethod, 0),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: obviously this should be reconsidered
	}
	if opts.SSHPassword != "" {
		log.Debug("Using SSH password authentication")
		config.Auth = append(config.Auth, ssh.Password(opts.SSHPassword))
	}
	if opts.SSHKeyFile != "" {
		log.Debug("Using SSH pubkey authentication")
		log.Debugf("Loading SSH key from %q\n", opts.SSHKeyFile)
		keyBytes, err := ioutil.ReadFile(opts.SSHKeyFile)
		if err != nil {
			return nil, err
		}
		key, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, err
		}
		config.Auth = append(config.Auth, ssh.PublicKeys(key))
	}

	addr := net.JoinHostPort(opts.Address, strconv.Itoa(opts.SSHPort))
	log.Debugf("Creating SSH connection with %s over TCP\n", addr)
	return ssh.Dial("tcp", addr, config)
}
