package types

import "io"

// ImageDownloader is an interface for pulling OCI container images and exporting
// them to tar archives or deployable registries. It can be implemented by different
// runtimes such as docker, containerd, podman, etc.
type ImageDownloader interface {
	// SaveImages will return a reader containing the contents of the exported
	// images provided as arguments.
	SaveImages(images []string, arch string, pullPolicy PullPolicy) (io.ReadCloser, error)
	// BuildRegistry will build a container registry with the given images and return a
	// a reader to a container image holding the backed up contents. It will be unpacked into
	// a running registry with auto-generated TLS at installation time.
	BuildRegistry(*BuildRegistryOptions) (io.ReadCloser, error)
}
