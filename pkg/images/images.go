package images

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

// NewImageDownloader returns a new interface for downloading and exporting container
// images.
func NewImageDownloader() types.ImageDownloader {
	return &dockerImageDownloader{}
}

type dockerImageDownloader struct{}

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

func (d *dockerImageDownloader) BuildRegistry(images []string, arch string, pullPolicy types.PullPolicy) ([]*types.Artifact, error) {
	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	// Create a manifest for the registry
	log.Info("Generating kubernetes manifests for the private registry")
	var buf bytes.Buffer
	err = registryTmpl.Execute(&buf, nil)
	if err != nil {
		return nil, err
	}
	body := buf.Bytes()
	deploymentManifest := &types.Artifact{
		Type: types.ArtifactManifest,
		Name: "private-registry-deployment.yaml",
		Body: ioutil.NopCloser(bytes.NewReader(body)),
		Size: int64(len(body)),
	}

	log.Info("Starting local private image registry")
	if err := ensureImagePulled(cli, "registry:2", arch, pullPolicy); err != nil {
		return nil, err
	}
	registryContainerConfig, registryHostConfig := registryContainerConfigs()
	log.Debugf("Registry container config: %+v\n", registryContainerConfig)
	log.Debugf("Registry host config: %+v\n", registryHostConfig)
	registryID, err := createAndStartContainer(cli, registryContainerConfig, registryHostConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cli.ContainerRemove(context.TODO(), registryID, dockertypes.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		}); err != nil {
			log.Warning("Error removing registry container:", err)
		}
	}()

	// Fetch the host port that was bound to the registry
	log.Debugf("Inspecting container %s for exposed registry port\n", registryID)
	localPort, err := getHostPortForContainer(cli, registryID, "5000/tcp")
	if err != nil {
		return nil, err
	}
	log.Debugf("Local private registry is exposed on port %s\n", localPort)

	// Proxy images into the registry
	images = sanitizeImageNameSlice(images)
	for _, image := range images {
		if err := ensureImagePulled(cli, image, arch, pullPolicy); err != nil {
			return nil, err
		}
		log.Infof("Pushing %s to private registry\n", image)
		localImageName := fmt.Sprintf("localhost:%s/%s", localPort, image)
		log.Debug("Using local image name", localImageName)
		if err := cli.ImageTag(context.TODO(), image, localImageName); err != nil {
			return nil, err
		}
		rdr, err := cli.ImagePush(context.TODO(), localImageName, dockertypes.ImagePushOptions{
			All:          true,
			RegistryAuth: "fake", // https://github.com/moby/moby/issues/10983
		})
		if err != nil {
			return nil, err
		}
		log.LevelReader(log.LevelDebug, rdr)
	}

	// Mount registry volumes into busybox image to take backup and commit the contents
	// to an image that can be used as an init container.
	log.Info("Exporting private registry contents to container image")
	if err := ensureImagePulled(cli, "busybox", arch, pullPolicy); err != nil {
		return nil, err
	}
	busyboxConfig, busyboxHostConfig := registryVolumeContainerConfigs(registryID)
	log.Debugf("Busybox container config: %+v\n", busyboxConfig)
	log.Debugf("Busybox host config: %+v\n", busyboxHostConfig)
	volContainerID, err := createAndStartContainer(cli, busyboxConfig, busyboxHostConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := cli.ContainerRemove(context.TODO(), volContainerID, dockertypes.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		}); err != nil {
			log.Warning("Error removing registry volume container:", err)
		}
	}()
	// Wait for the tar process to finish
	log.Debug("Waiting for container process to finish")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // make configurable
	defer cancel()
	statusCh, errCh := cli.ContainerWait(ctx, volContainerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	case res := <-statusCh:
		logs, err := cli.ContainerLogs(context.TODO(), volContainerID, dockertypes.ContainerLogsOptions{
			ShowStdout: true, ShowStderr: true,
		})
		if err != nil {
			log.Debug("Failed to retrieve container logs for", volContainerID, ":", err)
		} else {
			defer logs.Close()
			log.Debug("Container logs for", volContainerID)
			log.LevelReader(log.LevelDebug, logs)
		}
		if res.StatusCode != 0 {
			return nil, errors.New("Registry data backup exited with non-zero status code")
		}
	}

	// Commit the registry volume container to an image
	_, err = cli.ContainerCommit(context.TODO(), volContainerID, dockertypes.ContainerCommitOptions{Reference: "private-registry-data:latest"})
	if err != nil {
		return nil, err
	}

	// Save registry images
	rdr, err := cli.ImageSave(context.TODO(), []string{"private-registry-data:latest", "registry:2"})
	if err != nil {
		return nil, err
	}

	// Create artifact for registry images
	registryArtifact, err := util.ArtifactFromReader(types.ArtifactImages, "private-registry.tar", rdr)
	if err != nil {
		return nil, err
	}

	return []*types.Artifact{deploymentManifest, registryArtifact}, err
}
