package types

// Installer is an interface for laying a package manifest down on a system
// and setting up K3s.
type Installer interface {
	Install(opts *InstallOptions) error
}

// K3sRole represents the different roles a machine can take in the cluster
type K3sRole string

const (
	// K3sRoleServer represents a server instance in the control-plane
	K3sRoleServer K3sRole = "server"
	// K3sRoleAgent represents a worker node instance
	K3sRoleAgent K3sRole = "agent"
)

// InstallOptions are options to pass to an installation
type InstallOptions struct {
	// The path to the tar archive
	TarPath string
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
	// Whether to run as a server or agent\
	K3sRole K3sRole
}
