package cluster

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"

	v1 "github.com/tinyzimmer/k3p/pkg/build/package/v1"
	"github.com/tinyzimmer/k3p/pkg/cluster/node"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// New returns a new ClusterManager instance.
func New() types.ClusterManager { return &manager{} }

type manager struct{}

func (m *manager) RemoveNode(opts *types.RemoveNodeOptions) error { return nil }

func (m *manager) AddNode(opts *types.AddNodeOptions) error {

	var remoteAddr string
	var leader types.Node
	var err error

	if opts.RemoteLeader != "" {
		remoteAddr = opts.RemoteLeader
		connectOpts := *opts.NodeConnectOptions
		connectOpts.Address = remoteAddr
		leader, err = node.Connect(&connectOpts)
		if err != nil {
			return err
		}
	} else {
		log.Info("Determining current k3s external listening address")
		remoteAddr, err = getExternalK3sAddr()
		if err != nil {
			return err
		}
		log.Debug("K3s is listening on", remoteAddr)
		leader = node.Local()
	}

	// The reason we send the manifest over in pieces is because I was having strange bugs
	// with trying to send it over with the k3p binary and extract on the remote host.
	//
	// The tarball was moving over, but then ended up being an empty file on the other end.
	// Loading it locally and sending it in pieces works for now.
	log.Info("Loading package manifest")
	f, err := leader.GetFile(types.InstalledPackageFile)
	if err != nil {
		return err
	}
	pkg, err := v1.Load(f)
	if err != nil {
		return err
	}
	defer pkg.Close()

	var tokenRdr io.ReadCloser
	switch opts.NodeRole {
	case types.K3sRoleServer:
		log.Debug("Reading server join token from", types.ServerTokenFile)
		tokenRdr, err = leader.GetFile(types.ServerTokenFile)
	case types.K3sRoleAgent:
		log.Debug("Reading agent join token from", types.AgentTokenFile)
		tokenRdr, err = leader.GetFile(types.AgentTokenFile)
	default:
		return fmt.Errorf("Invalid node role %s", opts.NodeRole)
	}
	if err != nil {
		return err
	}
	defer tokenRdr.Close()

	token, err := ioutil.ReadAll(tokenRdr)
	if err != nil {
		return err
	}
	tokenStr := string(token)

	log.Infof("Connecting to server %s on port %d\n", opts.Address, opts.SSHPort)
	newNode, err := node.Connect(opts.NodeConnectOptions)
	if err != nil {
		return err
	}
	defer newNode.Close()

	if err := util.SyncPackageToNode(newNode, pkg); err != nil {
		return err
	}

	log.Infof("Joining instance as a new %s\n", opts.NodeRole)
	cmd := buildInstallCmd(remoteAddr, tokenStr, string(opts.NodeRole))
	log.Debug("Executing command on remote:", strings.Replace(cmd, tokenStr, "<redacted>", -1))
	return newNode.Execute(cmd, "K3S")
}

func buildInstallCmd(remoteAddr, token, nodeRole string) string {
	installCmd := fmt.Sprintf(
		`sudo sh -c 'INSTALL_K3S_SKIP_DOWNLOAD="true" K3S_URL="https://%s:6443" K3S_TOKEN="%s" %s %s'`,
		remoteAddr, token, path.Join(types.K3sScriptsDir, "install.sh"), nodeRole,
	)
	return installCmd
}
