package types

// VersionLatest is a string signaling that the latest version should be retrieved for k3s.
const VersionLatest string = "latest"

// ManifestMetaFile is the name used when writing the version information to an archive.
const ManifestMetaFile = "manifest.json"

// ManifestEULAFile is the name used when archiving an EULA.
const ManifestEULAFile = "EULA.txt"

// ManifestUserImagesFile is the name of the tarball where detected images are stored in an archive.
const ManifestUserImagesFile = "manifest-images.tar"

// K3sRootConfigDir is the root directory where k3s assets are stored
const K3sRootConfigDir = "/var/lib/rancher/k3s"

// ServerTokenFile is the file where the secret token is written for joining
// new control-plane instances in an HA setup.
const ServerTokenFile = "/var/lib/rancher/k3s/server/server-token"

// AgentTokenFile is the file where the secret token is written for joining
// new agents to the cluster
const AgentTokenFile = "/var/lib/rancher/k3s/server/node-token"

// InstalledPackageFile is the file where the original tarball is copied
// during the installation.
const InstalledPackageFile = "/var/lib/rancher/k3s/data/k3p-package.tar"

// InstalledConfigFile is the file where the variables used at installation are stored.
const InstalledConfigFile = "/var/lib/rancher/k3s/data/k3p-config.json"

// K3sManifestsDir is the directory where manifests are installed for k3s to pre-load on boot.
const K3sManifestsDir = "/var/lib/rancher/k3s/server/manifests"

// K3sImagesDir is the directory where images are pre-loaded on a server or agent.
const K3sImagesDir = "/var/lib/rancher/k3s/agent/images"

// K3sStaticDir is the directory where static content can be served from the k8s api.
const K3sStaticDir = "/var/lib/rancher/k3s/server/static/k3p"

// K3sScriptsDir is the directory where scripts are installed to the system.
const K3sScriptsDir = "/usr/local/bin/k3p-scripts"

// K3sBinDir is the directory where binaries are installed to the system.
const K3sBinDir = "/usr/local/bin"

// K3sEtcDir is the directory where configuration files are stored for k3s.
const K3sEtcDir = "/etc/rancher/k3s"

// K3sRegistriesYamlPath is the path where the k3s containerd configuration is stored.
const K3sRegistriesYamlPath = "/etc/rancher/k3s/registries.yaml"

// K3sKubeconfig is the path where the admin kubeconfig is stored on the system.
const K3sKubeconfig = "/etc/rancher/k3s/k3s.yaml"

// K3sInternalIPLabel is the label K3s uses for the internal IP of a node.
const K3sInternalIPLabel = "k3s.io/internal-ip"

// K3pManagedDockerLabel is the label placed on resources to mark that they were created by k3p.
const K3pManagedDockerLabel = "k3p.io/managed"

// K3pDockerClusterLabel is the label placed on k3p docker assets containing the cluster name.
const K3pDockerClusterLabel = "k3p.io/cluster-name"

// K3pDockerNodeNameLabel is the label placed on k3p docker assets containing the node name.
const K3pDockerNodeNameLabel = "k3p.io/node-name"

// K3pDockerNodeRoleLabel is the label where the node role is placed.
const K3pDockerNodeRoleLabel = "k3p.io/node-role"

// DefaultRegistryPort is the default node port used when a package includes a private registry.
const DefaultRegistryPort = 30100

// K3sRole represents the different roles a machine can take in the cluster
type K3sRole string

const (
	// K3sRoleServer represents a server instance in the control-plane
	K3sRoleServer K3sRole = "server"
	// K3sRoleAgent represents a worker node instance
	K3sRoleAgent K3sRole = "agent"
	// K3sRoleLoadBalancer is a special role used for running packages in docker containers
	K3sRoleLoadBalancer K3sRole = "loadbalancer"
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
	// ArtifactStatic represents static content to be hosted by the api server.
	ArtifactStatic ArtifactType = "static"
	// ArtifactEULA represents an End User License Agreement.
	ArtifactEULA ArtifactType = "eula"
	// ArtifactEtc is an artifact to be placed in /etc/rancher/k3s.
	ArtifactEtc ArtifactType = "etc"
)

// ImageBundleFormat declares how the images were bundled in a package. Currently
// either via raw tar balls, or a pre-loaded private registry.
type ImageBundleFormat string

const (
	// ImageBundleTar represents raw image tarballs.
	ImageBundleTar ImageBundleFormat = "raw"
	// ImageBundleRegistry represents a pre-loaded private registry.
	ImageBundleRegistry ImageBundleFormat = "registry"
)
