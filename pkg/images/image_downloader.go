package images

import "github.com/tinyzimmer/k3p/pkg/types"

// NewImageDownloader returns a new interface for downloading and exporting container
// images.
func NewImageDownloader() types.ImageDownloader {
	return &dockerImageDownloader{}
}

type dockerImageDownloader struct{}
