package node

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path"
	"strconv"

	"github.com/bramvdbogaerde/go-scp"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"golang.org/x/crypto/ssh"
)

// Connect will connect to a node over SSH with the given options.
func Connect(opts *types.NodeConnectOptions) (types.Node, error) {
	var err error
	n := &remoteNode{remoteAddr: opts.Address}
	n.client, err = getSSHClient(opts)
	return n, err
}

type remoteNode struct {
	client     *ssh.Client
	remoteAddr string
}

func (n *remoteNode) GetType() types.NodeType { return types.NodeRemote }

func (n *remoteNode) scpClient() (*scp.Client, error) {
	scpClient, err := scp.NewClientBySSH(n.client)
	if err != nil {
		return nil, err
	}
	scpClient.RemoteBinary = "sudo scp"
	return &scpClient, nil
}

type remoteReadCloser struct {
	sess *ssh.Session
	pipe io.Reader
}

func (r *remoteReadCloser) Read(p []byte) (int, error) { return r.pipe.Read(p) }

func (r *remoteReadCloser) Close() error {
	if err := r.sess.Wait(); err != nil {
		return err
	}
	return r.sess.Close()
}

func (n *remoteNode) GetFile(path string) (io.ReadCloser, error) {
	sess, err := n.client.NewSession()
	if err != nil {
		return nil, err
	}
	outPipe, err := sess.StdoutPipe()
	if err != nil {
		return nil, err
	}
	remoteRdr := &remoteReadCloser{sess: sess, pipe: outPipe}
	cmd := fmt.Sprintf("sudo cat %q", path)
	log.Debugf("Running command on %s: %s\n", n.remoteAddr, cmd)
	if err := sess.Start(cmd); err != nil {
		if cerr := sess.Close(); cerr != nil {
			log.Error("Unexpected error while closing failed ssh get file:", cerr)
		}
		return nil, err
	}
	return remoteRdr, nil
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
	log.Debugf("Sending %d bytes of %q to %q on %s and setting mode to %s\n", size, path.Base(destination), destination, n.remoteAddr, mode)
	return scpClient.Copy(rdr, destination, mode, size)
}

func (n *remoteNode) MkdirAll(dir string) error {
	sess, err := n.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	cmd := fmt.Sprintf("sudo mkdir -p %s", dir)
	log.Debugf("Running command on %s: %s\n", n.remoteAddr, cmd)
	return sess.Run(cmd)
}

func (n *remoteNode) Execute(opts *types.ExecuteOptions) error {
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
	cmd := buildCmdFromExecOpts(opts)
	log.Debugf("Executing command on %s: %s\n", n.remoteAddr, redactSecrets(cmd, opts.Secrets))
	go log.TailReader(opts.LogPrefix, outPipe)
	go log.TailReader(opts.LogPrefix, errPipe)
	return sess.Run(cmd)
}

func (n *remoteNode) GetK3sAddress() (string, error) {
	// the address is assumed to be the one we connected on (TODO: fix probably)
	return n.remoteAddr, nil
}

func (n *remoteNode) Close() error { return n.client.Close() }

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
