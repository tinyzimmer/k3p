package images

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

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

func (d *dockerImageDownloader) PullImages(images []string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	for _, name := range images {
		reader, err := cli.ImagePull(context.TODO(), name, dockertypes.ImagePullOptions{})
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			statusJSON := map[string]interface{}{}
			if err := json.Unmarshal(scanner.Bytes(), &statusJSON); err != nil {
				return err
			}
			statusStr, ok := statusJSON["status"]
			if !ok {
				continue
			}
			if strings.HasPrefix(statusStr.(string), "Pulling from") {
				var id string
				id, ok = statusJSON["id"].(string) // probably not okay
				if !ok {
					id = "<unknown>"
				}
				log.Println(fmt.Sprintf("%s:%s", statusStr, id))
				continue
			}
			// cant decide if i really want to invest that much effort into pretty output
			// also not being clear to the user when image already exists locally (should technically
			// check that condition instead of pulling first, or make behavior configurable)
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			return err
		}
	}

	return nil
}

func (d *dockerImageDownloader) SaveImagesTo(images []string, dest string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	reader, err := cli.ImageSave(context.TODO(), images)
	if err != nil {
		return err
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer reader.Close()
	defer out.Close()
	if _, err := io.Copy(out, reader); err != nil {
		return err
	}
	return nil
}
