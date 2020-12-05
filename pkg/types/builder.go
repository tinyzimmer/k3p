package types

// Builder is an interface for building application bundles to be distributed to systems.
type Builder interface {
	Build(*BuildOptions) error
}

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
	// A path to an optional file of newline delimited container images to include in the package
	ImageFile string
	// The directory to scan for kubernetes manifests and helm charts
	ManifestDir string
	// Arguments to pass to helm charts bundled with the application
	HelmArgs string
	// A list of directories to exclude while searching for manifests
	Excludes []string
	// The path to write the final archive to
	Output string
}
