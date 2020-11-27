package types

import "io"

// ImageDownloader is an interface for pulling OCI container images and exporting
// them to tar archives. It can be implemented by different runtimes such as docker,
// containerd, podman, etc.
type ImageDownloader interface {
	PullImages(images []string) error
	SaveImages(images []string) (io.ReadCloser, error)
}
