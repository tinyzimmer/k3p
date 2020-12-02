package cluster

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

// New returns a new ClusterManager instance.
func New() types.ClusterManager { return &manager{} }

type manager struct{}

func (m *manager) AddNode(opts *types.AddNodeOptions) error {
	// this function assumes port 6443, which is the default
	//
	// need to double check if this is actually configurable
	log.Info("Determining current k3s external listening address")
	remoteAddr, err := getExternalK3sAddr()
	if err != nil {
		return err
	}
	log.Debug("K3s is listening on", remoteAddr)

	log.Infof("Connecting to server %s on port %d", opts.NodeAddress, opts.SSHPort)
	client, err := getSSHClient(opts)
	if err != nil {
		return err
	}
	defer client.Close()

	log.Info("Copying package manifest to the new node")
	if err := sshSyncFile(client, types.InstalledPackageFile); err != nil {
		return err
	}

	log.Info("Copying the k3p binary to the new node")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	if err := sshSyncFile(client, ex); err != nil {
		return err
	}

	var token []byte
	switch opts.NodeRole {
	case types.K3sRoleServer:
		log.Debug("Reading server join token from", types.ServerTokenFile)
		token, err = ioutil.ReadFile(types.ServerTokenFile)
	case types.K3sRoleAgent:
		log.Debug("Reading agent join token from", types.AgentTokenFile)
		token, err = ioutil.ReadFile(types.AgentTokenFile)
	default:
		return fmt.Errorf("Invalid node role %s", opts.NodeRole)
	}

	if err != nil {
		return err
	}

	log.Infof("Joining new %s instance at %s", types.K3sRoleServer, opts.NodeAddress)
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	outPipe, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	go io.Copy(os.Stdout, outPipe)

	cmd := fmt.Sprintf("sudo %s install %s --join https://%s:6443 --join-role %s --join-token %s",
		ex, types.InstalledPackageFile, remoteAddr, string(opts.NodeRole), string(token),
	)
	log.Debug("Executing command on remote:", strings.Replace(cmd, string(token), "<redacted>", -1))
	return session.Run(cmd)
}
