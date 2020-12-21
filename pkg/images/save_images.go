package images

import (
	"context"
	"io"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

func (d *dockerImageDownloader) SaveImages(images []string, arch string, pullPolicy types.PullPolicy) (io.ReadCloser, error) {
	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	images = sanitizeImageNameSlice(images)
	for _, image := range images {
		if err := ensureImagePulled(cli, image, arch, pullPolicy); err != nil {
			return nil, err
		}
	}

	log.Debug("Saving images:", images)
	return cli.ImageSave(context.TODO(), images)
}
