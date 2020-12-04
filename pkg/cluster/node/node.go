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

// Node is an interface for interacting with a remote instance over SSH.
type Node interface {
	MkdirAll(dir string) error
	WriteFile(rdr io.ReadCloser, destination string, mode string, size int64) error
	Execute(cmd string, logPrefix string) error
	Close() error
}

// Connect will connect to a node with the given options.
func Connect(opts *types.NodeConnectOptions) (Node, error) {
	var err error
	n := &node{}
	n.client, err = getSSHClient(opts)
	return n, err
}

type node struct {
	client *ssh.Client
}

func (n *node) scpClient() (*scp.Client, error) {
	scpClient, err := scp.NewClientBySSH(n.client)
	if err != nil {
		return nil, err
	}
	scpClient.RemoteBinary = "sudo scp"
	return &scpClient, nil
}

func (n *node) WriteFile(rdr io.ReadCloser, destination string, mode string, size int64) error {
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

func (n *node) MkdirAll(dir string) error {
	sess, err := n.client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	cmd := fmt.Sprintf("sudo mkdir -p %s", dir)
	log.Debug("Running command on remote:", cmd)
	return sess.Run(cmd)
}

func (n *node) Execute(cmd string, logPrefix string) error {
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

func (n *node) Close() error { return n.client.Close() }

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
