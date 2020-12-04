package cluster

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	v1 "github.com/tinyzimmer/k3p/pkg/build/archive/v1"
	"github.com/tinyzimmer/k3p/pkg/cluster/node"
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

	// The reason we send the manifest over in pieces is because I was having strange bugs
	// with trying to send it over with the k3p binary and extract on the remote host.
	//
	// The tarball was moving over, but then ended up being an empty file on the other end.
	// Loading it locally and sending it in pieces works for now.
	log.Info("Loading package manifest")
	pkg, err := v1.Load(types.InstalledPackageFile)
	if err != nil {
		return err
	}
	defer pkg.Close()

	manifest, err := pkg.GetManifest()
	if err != nil {
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
	tokenStr := string(token)

	log.Infof("Connecting to server %s on port %d\n", opts.Address, opts.SSHPort)
	newNode, err := node.Connect(opts.NodeConnectOptions)
	if err != nil {
		return err
	}
	defer newNode.Close()

	if err := syncManifestToNode(newNode, manifest); err != nil {
		return err
	}

	log.Infof("Joining instance as a new %s\n", opts.NodeRole)
	cmd := buildInstallCmd(remoteAddr, tokenStr, string(opts.NodeRole))
	log.Debug("Executing command on remote:", strings.Replace(cmd, tokenStr, "<redacted>", -1))
	return newNode.Execute(cmd, "K3S")
}

func syncManifestToNode(remote node.Node, manifest *types.PackageManifest) error {
	log.Info("Installing binaries to remote machine at", types.K3sBinDir)
	for _, bin := range manifest.Bins {
		if err := remote.WriteFile(bin.Body, path.Join(types.K3sBinDir, bin.Name), "0755", bin.Size); err != nil {
			return err
		}
	}

	log.Info("Installing scripts to remote machine at", types.K3sScriptsDir)
	for _, script := range manifest.Scripts {
		if err := remote.WriteFile(script.Body, path.Join(types.K3sScriptsDir, script.Name), "0755", script.Size); err != nil {
			return err
		}
	}

	log.Info("Installing images to remote machine at", types.K3sImagesDir)
	for _, imgs := range manifest.Images {
		if err := remote.WriteFile(imgs.Body, path.Join(types.K3sImagesDir, imgs.Name), "0644", imgs.Size); err != nil {
			return err
		}
	}

	log.Info("Installing manifests to remote machine at", types.K3sManifestsDir)
	for _, mani := range manifest.Manifests {
		if err := remote.WriteFile(mani.Body, path.Join(types.K3sManifestsDir, mani.Name), "0644", mani.Size); err != nil {
			return err
		}
	}
	return nil
}

func buildInstallCmd(remoteAddr, token, nodeRole string) string {
	installCmd := fmt.Sprintf(
		`sudo sh -c 'INSTALL_K3S_SKIP_DOWNLOAD="true" K3S_URL="https://%s:6443" K3S_TOKEN="%s" %s %s'`,
		remoteAddr, token, path.Join(types.K3sScriptsDir, "install.sh"), nodeRole,
	)
	return installCmd
}
