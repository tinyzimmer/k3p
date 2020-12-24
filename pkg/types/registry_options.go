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

// RegistryTLSOptions repsent options to use when generating TLS secrets for an in-cluster
// private registry.
type RegistryTLSOptions struct {
	// A name to use when generating self-signed certificates
	Name string
	// The path to a TLS certificate  to use for the private registry. If left unset a
	// self-signed certificate chain is generated.
	RegistryTLSCertFile string
	// The path to an unencrypted TLS private key to use for the private registry that matches
	// the leaf certificate provided to RegistryTLSBundle. A key is generated if not provided.
	RegistryTLSKeyFile string
	// The path to the CA bundle for the provided TLS certificate
	RegistryTLSCAFile string
}
