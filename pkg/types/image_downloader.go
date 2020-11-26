package types

// ImageDownloader is an interface for pulling OCI container images and exporting
// them to tar archives. It can be implemented by different runtimes such as docker,
// containerd, podman, etc.
type ImageDownloader interface {
	PullImages(images []string) error
	SaveImagesTo(images []string, dest string) error
}
