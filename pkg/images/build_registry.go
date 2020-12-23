package images

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"github.com/tinyzimmer/k3p/pkg/images/registry"
	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var requiredRegistryImages = []string{"registry:2", "busybox", registry.KubenabImage}

func setOptDefaults(opts *types.BuildRegistryOptions) *types.BuildRegistryOptions {
	if opts.AppVersion == "" {
		opts.AppVersion = types.VersionLatest
	}

	if opts.PullPolicy == "" {
		opts.PullPolicy = types.PullPolicyAlways
	}

	return opts
}

func (d *dockerImageDownloader) BuildRegistry(opts *types.BuildRegistryOptions) (io.ReadCloser, error) {
	opts = setOptDefaults(opts)

	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	// Ensure all needed images are present
	userImages := sanitizeImageNameSlice(opts.Images)
	for _, img := range append(requiredRegistryImages, userImages...) {
		if err := ensureImagePulled(cli, img, opts.Arch, opts.PullPolicy); err != nil {
			return nil, err
		}
	}

	log.Info("Starting local private image registry")
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
	if err := waitForLocalRegistry(localPort, time.Second*10); err != nil {
		return nil, err
	}

	// Proxy user images into the registry
	for _, image := range userImages {
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
	_, err = cli.ContainerCommit(context.TODO(), volContainerID, dockertypes.ContainerCommitOptions{Reference: opts.RegistryImageName()})
	if err != nil {
		return nil, err
	}

	// Save all images for the registry
	return cli.ImageSave(context.TODO(), append(requiredRegistryImages, opts.RegistryImageName()))
}
