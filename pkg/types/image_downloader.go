package types

import "io"

// ImageDownloader is an interface for pulling OCI container images and exporting
// them to tar archives. It can be implemented by different runtimes such as docker,
// containerd, podman, etc.
type ImageDownloader interface {
	// PullImages should return a reader containing the contents of the exported
	// images provided as arguments.
	PullImages(images []string, arch string) (io.ReadCloser, error)
}
