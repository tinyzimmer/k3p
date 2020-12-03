package cluster

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	v1 "github.com/tinyzimmer/k3p/pkg/build/archive/v1"
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
	defer pkg.Close()
	if err != nil {
		return err
	}
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

	log.Infof("Connecting to server %s on port %d", opts.NodeAddress, opts.SSHPort)
	client, err := getSSHClient(opts)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := sshSyncManifest(client, manifest); err != nil {
		return err
	}

	log.Infof("Joining instance as a new %s", opts.NodeRole)
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	outPipe, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	errPipe, err := session.StderrPipe()
	if err != nil {
		return err
	}
	go io.Copy(os.Stdout, outPipe)
	go io.Copy(os.Stdout, errPipe)
	cmd := buildInstallCmd(remoteAddr, tokenStr, string(opts.NodeRole))
	log.Debug("Executing command on remote:", strings.Replace(cmd, tokenStr, "<redacted>", -1))
	return session.Run(cmd)
}

func buildInstallCmd(remoteAddr, token, nodeRole string) string {
	installCmd := fmt.Sprintf(
		`/bin/sh -c 'INSTALL_K3S_SKIP_DOWNLOAD="true" K3S_URL="https://%s:6443" K3S_TOKEN="%s" %s %s'`,
		remoteAddr, token, path.Join(types.K3sScriptsDir, "install.sh"), nodeRole,
	)
	return installCmd
}
