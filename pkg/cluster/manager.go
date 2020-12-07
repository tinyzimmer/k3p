package cluster

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strings"

	v1 "github.com/tinyzimmer/k3p/pkg/build/package/v1"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// New returns a new ClusterManager instance.
func New(leader types.Node) types.ClusterManager { return &manager{leader: leader} }

type manager struct{ leader types.Node }

func (m *manager) RemoveNode(opts *types.RemoveNodeOptions) error { return nil }

func (m *manager) AddNode(newNode types.Node, opts *types.AddNodeOptions) error {

	remoteAddr, err := m.leader.GetK3sAddress()
	if err != nil {
		return err
	}
	log.Debug("K3s is listening on", remoteAddr)

	// The reason we send the manifest over in pieces is because I was having strange bugs
	// with trying to send it over with the k3p binary and extract on the remote host.
	//
	// The tarball was moving over, but then ended up being an empty file on the other end.
	// Loading it locally and sending it in pieces works for now.
	log.Info("Loading package manifest")
	f, err := m.leader.GetFile(types.InstalledPackageFile)
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
		tokenRdr, err = m.leader.GetFile(types.ServerTokenFile)
	case types.K3sRoleAgent:
		log.Debug("Reading agent join token from", types.AgentTokenFile)
		tokenRdr, err = m.leader.GetFile(types.AgentTokenFile)
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
	tokenStr := strings.TrimSpace(string(token))

	if err := util.SyncPackageToNode(newNode, pkg); err != nil {
		return err
	}

	log.Infof("Joining instance as a new %s\n", opts.NodeRole)
	execOpts := buildInstallOpts(remoteAddr, tokenStr, string(opts.NodeRole))
	return newNode.Execute(execOpts)
}

func buildInstallOpts(remoteAddr, token, nodeRole string) *types.ExecuteOptions {
	return &types.ExecuteOptions{
		Env: map[string]string{
			"INSTALL_K3S_SKIP_DOWNLOAD": "true",
			"K3S_URL":                   fmt.Sprintf("https://%s:6443", remoteAddr),
			"K3S_TOKEN":                 token,
		},
		Command:   fmt.Sprintf("sh %s %s", path.Join(types.K3sScriptsDir, "install.sh"), nodeRole),
		LogPrefix: "K3S",
		Secrets:   []string{token},
	}
}
