package cluster

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	v1 "github.com/tinyzimmer/k3p/pkg/build/package/v1"
	"github.com/tinyzimmer/k3p/pkg/cluster/kubernetes"
	"github.com/tinyzimmer/k3p/pkg/cluster/node"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// New returns a new ClusterManager instance.
func New(leader types.Node) types.ClusterManager { return &manager{leader: leader} }

type manager struct{ leader types.Node }

func (m *manager) getKubeconfig() ([]byte, error) {
	f, err := m.leader.GetFile(types.K3sKubeconfig)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	body, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	addr, err := m.leader.GetK3sAddress()
	if err != nil {
		return nil, err
	}
	return []byte(strings.Replace(string(body), "127.0.0.1", addr, 1)), nil
}

func (m *manager) RemoveNode(opts *types.RemoveNodeOptions) error {
	log.Debug("Retrieve kubeconfig from leader")
	cfg, err := m.getKubeconfig()
	if err != nil {
		return err
	}
	cli, err := kubernetes.New(cfg)
	if err != nil {
		return err
	}
	var nodeName string
	if opts.Name != "" {
		nodeName = opts.Name
	} else if opts.IPAddress != "" {
		log.Debug("Looking up node by IP", opts.IPAddress)
		node, err := cli.GetNodeByIP(opts.IPAddress)
		if err != nil {
			return err
		}
		nodeName = node.GetName()
	}

	if opts.Uninstall {
		if opts.IPAddress == "" {
			ip, err := cli.GetIPByNodeName(nodeName)
			if err != nil {
				return err
			}
			opts.NodeConnectOptions.Address = ip
		} else {
			opts.NodeConnectOptions.Address = opts.IPAddress
		}
	}

	log.Info("Deleting node", nodeName)
	if err := cli.RemoveNode(nodeName); err != nil {
		return err
	}

	// Wait for the node to be deleted
	log.Infof("Waiting for %q to be removed from the cluster\n", nodeName)
	var failCount int
ListNodesLoop:
	for {
		nodes, err := cli.ListNodes()
		if err != nil {
			if failCount > 3 {
				return err
			}
			log.Debug("Failure while listing nodes, retrying - error:", err.Error())
			failCount++
			continue ListNodesLoop
		}
		failCount = 0
		for _, node := range nodes {
			if node.GetName() == nodeName {
				log.Debug("Still waiting for node to be removed")
				time.Sleep(time.Second)
				continue ListNodesLoop
			}
		}
		break ListNodesLoop
	}

	if opts.Uninstall {
		log.Infof("Connecting to %s and uninstalling k3s\n", nodeName)
		oldNode, err := node.Connect(opts.NodeConnectOptions)
		if err != nil {
			return err
		}
		return oldNode.Execute(&types.ExecuteOptions{
			Command: "k3s-uninstall.sh", // TODO: type cast somewhere
		})
	}

	return nil
}

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
	defer f.Close()

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

	log.Debug("Loading installed package configuration")
	var installedConfig types.InstallConfig
	cfgFile, err := m.leader.GetFile(types.InstalledConfigFile)
	if err != nil {
		return err
	}
	defer cfgFile.Close()
	body, err := ioutil.ReadAll(cfgFile)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, &installedConfig); err != nil {
		return err
	}

	if err := util.SyncPackageToNode(newNode, pkg, &installedConfig); err != nil {
		return err
	}

	log.Infof("Joining instance as a new %s\n", opts.NodeRole)
	execOpts, err := buildInstallOpts(pkg, &installedConfig, remoteAddr, tokenStr, opts.NodeRole)
	if err != nil {
		return err
	}
	return newNode.Execute(execOpts)
}

func buildInstallOpts(pkg types.Package, cfg *types.InstallConfig, remoteAddr, token string, nodeRole types.K3sRole) (*types.ExecuteOptions, error) {
	opts := cfg.DeepCopy().InstallOptions
	pkgConf := pkg.GetMeta().DeepCopy().Sanitize().GetPackageConfig()
	if pkgConf != nil {
		if err := pkgConf.ApplyVariables(opts.Variables); err != nil {
			return nil, err
		}
	}
	opts.ServerURL = fmt.Sprintf("https://%s:%d", remoteAddr, cfg.InstallOptions.APIListenPort)
	opts.NodeToken = token
	opts.K3sRole = nodeRole
	return opts.ToExecOpts(pkgConf), nil
}
