package types

import "io"

// ImageDownloader is an interface for pulling OCI container images and exporting
// them to tar archives or deployable registries. It can be implemented by different
// runtimes such as docker, containerd, podman, etc.
type ImageDownloader interface {
	// SaveImages should return a reader containing the contents of the exported
	// images provided as arguments.
	SaveImages(images []string, arch string, pullPolicy PullPolicy) (io.ReadCloser, error)
	// BuildRegistry should build a container registry with the given images and return a
	// slice of artifacts to be bundled in a package. The artifacts should usually contain
	// a container image and manifest for launching it.
	BuildRegistry(*BuildRegistryOptions) ([]*Artifact, error)
}
