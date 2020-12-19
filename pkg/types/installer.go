package types

import (
	"fmt"
	"path"
	"strings"
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
	// The port that the k3s API server should listen on
	APIListenPort int
	// Extra arguments to pass to the k3s server or agent process that are not included
	// in the package.
	K3sServerArgs []string
	K3sAgentArgs  []string
	// Whether to run with --cluster-init
	InitHA bool
	// Whether to run as a server or agent
	K3sRole K3sRole
	// Variables contain substitutions to perform on manifests before
	// installing them to the system.
	Variables map[string]string
}

// DeepCopy creates a copy of these installation options.
func (opts *InstallOptions) DeepCopy() *InstallOptions {
	newOpts := &InstallOptions{
		NodeName:       opts.NodeName,
		AcceptEULA:     opts.AcceptEULA,
		ServerURL:      opts.ServerURL,
		NodeToken:      opts.NodeToken,
		ResolvConf:     opts.ResolvConf,
		KubeconfigMode: opts.KubeconfigMode,
		APIListenPort:  opts.APIListenPort,
		K3sServerArgs:  make([]string, len(opts.K3sServerArgs)),
		K3sAgentArgs:   make([]string, len(opts.K3sAgentArgs)),
		InitHA:         opts.InitHA,
		K3sRole:        opts.K3sRole,
		Variables:      make(map[string]string),
	}
	copy(newOpts.K3sServerArgs, opts.K3sServerArgs)
	copy(newOpts.K3sAgentArgs, opts.K3sAgentArgs)
	for k, v := range opts.Variables {
		newOpts.Variables[k] = v
	}
	return newOpts
}

// ToExecOpts converts these install options into execute options to pass to a
// node.
func (opts *InstallOptions) ToExecOpts(cfg *PackageConfig) *ExecuteOptions {
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

	var execFields []string
	switch opts.K3sRole {
	case K3sRoleServer, "":
		execFields = append([]string{string(K3sRoleServer)}, opts.K3sServerArgs...)
	case K3sRoleAgent:
		execFields = append([]string{string(K3sRoleAgent)}, opts.K3sAgentArgs...)
	}

	// Build out an exec string from the configuration
	if cfg != nil {
		switch opts.K3sRole {
		case K3sRoleServer, "":
			execFields = cfg.ServerArgs(execFields)
		case K3sRoleAgent:
			execFields = cfg.AgentArgs(execFields)
		}
	}

	if opts.APIListenPort != 0 && opts.K3sRole != K3sRoleAgent {
		execFields = append(execFields, fmt.Sprintf("--https-listen-port=%d", opts.APIListenPort))
	}

	if args := strings.Join(execFields, " "); args != "" {
		env["INSTALL_K3S_EXEC"] = args
	}

	secrets := []string{}
	if opts.NodeToken != "" {
		secrets = []string{opts.NodeToken}
	}

	return &ExecuteOptions{
		Env:     env,
		Command: fmt.Sprintf("sh %q", path.Join(K3sScriptsDir, "install.sh")),
		Secrets: secrets,
	}
}

// InstallConfig represents the values that were collected at installation time. It is used
// to serialize the configuration used to disk for future node-add/join operations.
type InstallConfig struct {
	// Options passed at installation
	InstallOptions *InstallOptions
}

// DeepCopy creates a copy of this InstallConfig.
func (i *InstallConfig) DeepCopy() *InstallConfig {
	return &InstallConfig{
		InstallOptions: i.InstallOptions.DeepCopy(),
	}
}
