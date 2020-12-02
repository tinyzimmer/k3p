package cluster

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strconv"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"golang.org/x/crypto/ssh"
)

func sshSyncFile(client *ssh.Client, file string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	cmd := fmt.Sprintf("mkdir -p %s", path.Dir(file))
	log.Debug("Executing command on remote:", cmd)
	if err := session.Run(cmd); err != nil {
		return err
	}
	session, err = client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	in, err := os.Open(file)
	if err != nil {
		return err
	}
	session.Stdin = in
	cmd = fmt.Sprintf("sudo tee %s", file)
	log.Debug("Executing command on remote:", cmd)
	return session.Run(cmd)
}

func getSSHClient(opts *types.AddNodeOptions) (*ssh.Client, error) {
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
		log.Debugf("Loading SSH key from %q", opts.SSHKeyFile)
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

	addr := net.JoinHostPort(opts.NodeAddress, strconv.Itoa(opts.SSHPort))
	log.Debugf("Creating SSH connection with %s over TCP", addr)
	return ssh.Dial("tcp", addr, config)
}
