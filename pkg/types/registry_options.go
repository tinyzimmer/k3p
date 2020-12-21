package types

// BuildRegistryOptions are options for configuring an in-cluster private
// container registry.
type BuildRegistryOptions struct {
	// A name to use when generating identifiers for various resources
	Name string
	// A list of images to bundle in the registry
	Images []string
	// Architecture to build the registry for
	Arch string
	// Pull policy to use while building the registry
	PullPolicy PullPolicy
	// The password to use for authentication to the registry, if this is blank one will
	// be generated.
	RegistrySecret string
}
