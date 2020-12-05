package types

// VersionLatest is a string signaling that the latest version should be retrieved for k3s.
const VersionLatest string = "latest"

// ManifestMetaFile is the name used when writing the version information to an archive.
const ManifestMetaFile = "manifest.json"

// ManifestEULAFile is the name used when archiving an EULA.
const ManifestEULAFile = "EULA.txt"

// ServerTokenFile is the file where the secret token is written for joining
// new control-plane instances in an HA setup.
const ServerTokenFile = "/var/lib/rancher/k3s/server/server-token"

// AgentTokenFile is the file where the secret token is written for joining
// new agents to the cluster
const AgentTokenFile = "/var/lib/rancher/k3s/server/node-token"

// InstalledPackageFile is the file where the original tarball is copied
// during the installation.
const InstalledPackageFile = "/var/lib/rancher/k3s/data/package.tar"

// K3sManifestsDir is the directory where manifests are installed for k3s to pre-load on boot.
const K3sManifestsDir = "/var/lib/rancher/k3s/server/manifests"

// K3sImagesDir is the directory where images are pre-loaded on a server or agent.
const K3sImagesDir = "/var/lib/rancher/k3s/agent/images"

// K3sScriptsDir is the directory where scripts are installed to the system.
const K3sScriptsDir = "/usr/local/bin/k3p-scripts"

// K3sBinDir is the directory where binaries are installed to the system.
const K3sBinDir = "/usr/local/bin"

// K3sKubeconfig is the path where the admin kubeconfig is stored on the system.
const K3sKubeconfig = "/etc/rancher/k3s/k3s.yaml"

// K3sInternalIPLabel is the label K3s uses for the internal IP of a node
const K3sInternalIPLabel = "k3s.io/internal-ip"

// K3sRole represents the different roles a machine can take in the cluster
type K3sRole string

const (
	// K3sRoleServer represents a server instance in the control-plane
	K3sRoleServer K3sRole = "server"
	// K3sRoleAgent represents a worker node instance
	K3sRoleAgent K3sRole = "agent"
)

// ArtifactType declares a type of artifact to be included in a bundle.
type ArtifactType string

const (
	// ArtifactBin represents a binary artifact.
	ArtifactBin ArtifactType = "bin"
	// ArtifactImages represents a container image artifact.
	ArtifactImages ArtifactType = "images"
	// ArtifactScript represents a script artifact.
	ArtifactScript ArtifactType = "script"
	// ArtifactManifest represents a kubernetes manifest artifact.
	ArtifactManifest ArtifactType = "manifest"
)
