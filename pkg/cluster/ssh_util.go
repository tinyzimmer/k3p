package cluster

import (
	"fmt"
	"io/ioutil"
	"net"
	"path"
	"strconv"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"

	scp "github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
)

func newScpClient(client *ssh.Client) (scp.Client, error) {
	scpClient, err := scp.NewClientBySSH(client)
	if err != nil {
		return scp.Client{}, err
	}
	scpClient.RemoteBinary = "sudo scp"
	return scpClient, nil
}

func sshSyncManifest(client *ssh.Client, manifest *types.PackageManifest) error {
	log.Info("Installing binaries to remote machine at", types.K3sBinDir)
	if err := sshMkdirAll(client, types.K3sBinDir); err != nil {
		return err
	}
	for _, bin := range manifest.Bins {
		defer bin.Body.Close()
		scpClient, err := newScpClient(client)
		if err != nil {
			return err
		}
		defer scpClient.Close()
		destPath := path.Join(types.K3sBinDir, bin.Name)
		log.Debugf("Sending %d bytes of %q to %q and setting mode to 0755", bin.Size, bin.Name, destPath)
		if err := scpClient.Copy(bin.Body, destPath, "0755", bin.Size); err != nil {
			return err
		}
	}

	log.Info("Installing scripts to remote machine at", types.K3sScriptsDir)
	if err := sshMkdirAll(client, types.K3sScriptsDir); err != nil {
		return err
	}
	for _, script := range manifest.Scripts {
		defer script.Body.Close()
		scpClient, err := newScpClient(client)
		if err != nil {
			return err
		}
		defer scpClient.Close()
		destPath := path.Join(types.K3sScriptsDir, script.Name)
		log.Debugf("Sending %d bytes of %q to %q and setting mode to 0755", script.Size, script.Name, destPath)
		if err := scpClient.Copy(script.Body, destPath, "0755", script.Size); err != nil {
			return err
		}
	}

	log.Info("Installing images to remote machine at", types.K3sImagesDir)
	if err := sshMkdirAll(client, types.K3sImagesDir); err != nil {
		return err
	}
	for _, imgs := range manifest.Images {
		defer imgs.Body.Close()
		scpClient, err := newScpClient(client)
		if err != nil {
			return err
		}
		defer scpClient.Close()
		destPath := path.Join(types.K3sImagesDir, imgs.Name)
		log.Debugf("Sending %d bytes of %q to %q and setting mode to 0644", imgs.Size, imgs.Name, destPath)
		if err := scpClient.Copy(imgs.Body, destPath, "0644", imgs.Size); err != nil {
			return err
		}
	}

	log.Info("Installing manifests to remote machine at", types.K3sManifestsDir)
	if err := sshMkdirAll(client, types.K3sManifestsDir); err != nil {
		return err
	}
	for _, mani := range manifest.Manifests {
		defer mani.Body.Close()
		if len(strings.Split(mani.Name, "/")) > 1 {
			base := path.Join(types.K3sManifestsDir, path.Dir(mani.Name))
			if err := sshMkdirAll(client, base); err != nil {
				return err
			}
		}
		scpClient, err := newScpClient(client)
		if err != nil {
			return err
		}
		defer scpClient.Close()
		destPath := path.Join(types.K3sManifestsDir, mani.Name)
		log.Debugf("Sending %d bytes of %q to %q and setting mode to 0644", mani.Size, mani.Name, destPath)
		if err := scpClient.Copy(mani.Body, destPath, "0644", mani.Size); err != nil {
			return err
		}
	}

	return nil
}

func sshMkdirAll(client *ssh.Client, dir string) error {
	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	cmd := fmt.Sprintf("sudo mkdir -p %s", dir)
	log.Debug("Running command on remote:", cmd)
	return sess.Run(cmd)
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
