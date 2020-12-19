package images

import (
	"context"
	"fmt"
	"io"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// NewImageDownloader returns a new interface for downloading and exporting container
// images.
func NewImageDownloader() types.ImageDownloader {
	return &dockerImageDownloader{}
}

type dockerImageDownloader struct{}

func (d *dockerImageDownloader) PullImages(images []string, arch string, pullPolicy types.PullPolicy) (io.ReadCloser, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		switch pullPolicy {
		case types.PullPolicyNever:
			imgs, err := cli.ImageList(context.TODO(), dockertypes.ImageListOptions{
				Filters: filters.NewArgs(filters.Arg("reference", image)),
			})
			if err != nil {
				return nil, err
			}
			if len(imgs) == 0 {
				return nil, fmt.Errorf("Image %s is not present on the machine", image)
			}
		case types.PullPolicyAlways:
			log.Infof("Pulling image for %s\n", image)
			rdr, err := cli.ImagePull(context.TODO(), image, dockertypes.ImagePullOptions{Platform: arch})
			if err != nil {
				return nil, err
			}
			log.LevelReader(log.LevelDebug, rdr)
		case types.PullPolicyIfNotPresent:
			imgs, err := cli.ImageList(context.TODO(), dockertypes.ImageListOptions{
				Filters: filters.NewArgs(filters.Arg("reference", image)),
			})
			if err != nil {
				return nil, err
			}
			if len(imgs) != 1 {
				log.Infof("Pulling image for %s\n", image)
				rdr, err := cli.ImagePull(context.TODO(), image, dockertypes.ImagePullOptions{Platform: arch})
				if err != nil {
					return nil, err
				}
				log.LevelReader(log.LevelDebug, rdr)
			} else {
				log.Infof("Image %s already present on the machine\n", image)
			}
		}

	}

	return cli.ImageSave(context.TODO(), images)
}
