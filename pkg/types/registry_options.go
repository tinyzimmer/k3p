package types

import "fmt"

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
}

// RegistryImageName returns the name to use for the image containing the registry contents.
func (opts *BuildRegistryOptions) RegistryImageName() string {
	return fmt.Sprintf("%s-private-registry-data:%s", opts.Name, opts.AppVersion)
}
