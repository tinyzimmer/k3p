package images

import (
	"context"
	"io"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// NewImageDownloader returns a new interface for downloading and exporting container
// images.
func NewImageDownloader() types.ImageDownloader {
	return &dockerImageDownloader{}
}

type dockerImageDownloader struct{}

func (d *dockerImageDownloader) PullImages(images []string, arch string) (io.ReadCloser, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		log.Infof("Pulling image for %s\n", image)
		rdr, err := cli.ImagePull(context.TODO(), image, dockertypes.ImagePullOptions{Platform: arch})
		if err != nil {
			return nil, err
		}
		log.DebugReader(rdr)
	}

	return cli.ImageSave(context.TODO(), images)
}
