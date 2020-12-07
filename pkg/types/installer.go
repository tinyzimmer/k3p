package types

import (
	"fmt"
	"path"
)

// Installer is an interface for laying a package manifest down on a system
// and setting up K3s.
type Installer interface {
	Install(node Node, pkg Package, opts *InstallOptions) error
}

// InstallOptions are options to pass to an installation
type InstallOptions struct {
	// An optional name to give the node
	NodeName string
	// Whether to skip viewing any EULA included in the package
	AcceptEULA bool
	// The URL to an already running k3s server to join as an agent
	ServerURL string
	// The node token from an already running k3s server
	NodeToken string
	// An optional resolv conf to use when configuring DNS
	ResolvConf string
	// Optionally override the default k3s kubeconfig mode (0600)
	// It is a string so it can be passed directly as an env var
	KubeconfigMode string
	// Extra arguments to pass to the k3s server or agent process
	K3sExecArgs string
	// Whether to run with --cluster-init
	InitHA bool
	// Whether to run as a server or agent
	K3sRole K3sRole
}

// ToExecOpts converts these install options into execute options to pass to a
// node.
func (opts *InstallOptions) ToExecOpts() *ExecuteOptions {
	env := map[string]string{
		"INSTALL_K3S_SKIP_DOWNLOAD": "true",
	}

	if opts.NodeName != "" {
		env["K3S_NODE_NAME"] = opts.NodeName
	}

	if opts.ResolvConf != "" {
		env["K3S_RESOLV_CONF"] = opts.ResolvConf
	}

	if opts.KubeconfigMode != "" {
		env["K3S_KUBECONFIG_MODE"] = opts.KubeconfigMode
	}

	if opts.NodeToken != "" {
		env["K3S_TOKEN"] = opts.NodeToken
	}

	if opts.ServerURL != "" {
		env["K3S_URL"] = opts.ServerURL
	}

	if opts.K3sExecArgs != "" {
		env["INSTALL_K3S_EXEC"] = opts.K3sExecArgs
	}

	secrets := []string{}
	if opts.NodeToken != "" {
		secrets = []string{opts.NodeToken}
	}

	return &ExecuteOptions{
		Env:       env,
		Command:   fmt.Sprintf("sh %q %s", path.Join(K3sScriptsDir, "install.sh"), string(opts.K3sRole)),
		LogPrefix: "K3S",
		Secrets:   secrets,
	}
}
