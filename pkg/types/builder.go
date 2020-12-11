package types

// Builder is an interface for building application bundles to be distributed to systems.
type Builder interface {
	Build(*BuildOptions) error
}

// PullPolicy represents the pull policy to use when bundling images
// TODO: This should probably be pulled from corev1.
type PullPolicy string

// Valid pull policies
const (
	PullPolicyAlways       PullPolicy = "always"
	PullPolicyNever        PullPolicy = "never"
	PullPolicyIfNotPresent PullPolicy = "ifnotpresent"
)

// BuildOptions is a struct containing options to pass to the build operation.
type BuildOptions struct {
	// The version of the package being built
	BuildVersion string
	// The name of the package, if not provided one is generated using docker's name generator
	Name string
	// The version of K3s to bundle with the package, overrides K3sChannel
	K3sVersion string
	// The release channel to retrieve the latest K3s version from
	K3sChannel string
	// The CPU architecture to target the package for
	Arch string
	// An optional EULA to provide with the package
	EULAFile string
	// An optional config file providing variables to be used at installation
	ConfigFile string
	// A path to an optional file of newline delimited container images to include in the package
	ImageFile string
	// A list of images to include in the package
	Images []string
	// The directory to scan for kubernetes manifests and helm charts
	ManifestDirs []string
	// Arguments to pass to helm charts bundled with the application
	HelmArgs string
	// A list of directories to exclude while searching for manifests
	Excludes []string
	// Don't bundle docker images with the archive
	ExcludeImages bool
	// The pull policy to use
	PullPolicy PullPolicy
	// The path to write the final archive to
	Output string
}
