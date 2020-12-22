package types

// BuildRegistryOptions are options for configuring an in-cluster private
// container registry.
type BuildRegistryOptions struct {
	// A name to use when generating identifiers for various resources
	Name string
	// The version of the application this registry is being built for.
	// Defaults to latest.
	AppVersion string
	// A list of images to bundle in the registry
	Images []string
	// Architecture to build the registry for
	Arch string
	// Pull policy to use while building the registry
	PullPolicy PullPolicy
	// The password to use for authentication to the registry, if this is blank one will
	// be generated.
	RegistrySecret string
	// The node port that the private registry will listen on when installed. Defaults to
	// 30100.
	RegistryNodePort string
}
