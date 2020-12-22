package images

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"golang.org/x/crypto/bcrypt"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
	"github.com/tinyzimmer/k3p/pkg/util"
)

const kubenabImage = "docker.bintray.io/kubenab:0.3.4"

var requiredRegistryImages = []string{"registry:2", "busybox", kubenabImage}

func setOptDefaults(opts *types.BuildRegistryOptions) *types.BuildRegistryOptions {
	if opts.RegistrySecret == "" {
		opts.RegistrySecret = util.GenerateToken(16)
	}

	if opts.AppVersion == "" {
		opts.AppVersion = types.VersionLatest
	}

	if opts.RegistryNodePort == "" {
		opts.RegistryNodePort = "30100"
	}

	if opts.PullPolicy == "" {
		opts.PullPolicy = types.PullPolicyAlways
	}

	return opts
}

func (d *dockerImageDownloader) BuildRegistry(opts *types.BuildRegistryOptions) ([]*types.Artifact, error) {
	opts = setOptDefaults(opts)

	cli, err := getDockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	regDataImgName := fmt.Sprintf("%s-private-registry-data:%s", opts.Name, opts.AppVersion)

	// Generate certificates for the registry
	// TODO: Allow user to supply certificates
	log.Info("Generating PKI for registry TLS")
	caCert, caPriv, err := generateCACertificate(opts.Name)
	if err != nil {
		return nil, err
	}
	registryCert, registryPriv, err := generateRegistryCertificate(caCert, caPriv, opts.Name)
	if err != nil {
		return nil, err
	}
	caCertPem, _, err := encodeToPEM(caCert, caPriv)
	if err != nil {
		return nil, err
	}
	registryCertPEM, registryKeyPEM, err := encodeToPEM(registryCert, registryPriv)
	if err != nil {
		return nil, err
	}

	caCertificate := &types.Artifact{
		Type: types.ArtifactEtc,
		Name: "registry-ca.crt",
		Body: ioutil.NopCloser(bytes.NewReader(caCertPem)),
		Size: int64(len(caCertPem)),
	}

	// Generate htpasswd file for the registry
	log.Info("Generating secrets for registry authentication")
	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(opts.RegistrySecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	htpasswd := append([]byte("registry:"), passwordBytes...)
	htpasswd = append(htpasswd, []byte("\n")...)

	// Create a manifest for the registry
	log.Info("Generating kubernetes manifests for the private registry")
	var buf bytes.Buffer
	err = registryTmpl.Execute(&buf, map[string]string{
		"TLSCertificate":       string(registryCertPEM),
		"TLSPrivateKey":        string(registryKeyPEM),
		"TLSCACertificate":     string(caCertPem),
		"RegistryAuthHtpasswd": string(htpasswd),
		"KubenabImage":         kubenabImage,
		"RegistryDataImage":    regDataImgName,
		"RegistryNodePort":     opts.RegistryNodePort,
	})
	if err != nil {
		return nil, err
	}
	body := buf.Bytes()
	deploymentManifest := &types.Artifact{
		Type: types.ArtifactManifest,
		Name: fmt.Sprintf("%s-private-registry-deployment.yaml", opts.Name),
		Body: ioutil.NopCloser(bytes.NewReader(body)),
		Size: int64(len(body)),
	}

	// Generate a registries.yaml
	var yamlBuf bytes.Buffer
	err = registriesYamlTmpl.Execute(&yamlBuf, map[string]string{
		"Username":         "registry",
		"Password":         opts.RegistrySecret,
		"RegistryNodePort": opts.RegistryNodePort,
	})
	if err != nil {
		return nil, err
	}
	registriesBody := yamlBuf.Bytes()
	registriesYamlArtifact := &types.Artifact{
		Type: types.ArtifactEtc,
		Name: "registries.yaml",
		Body: ioutil.NopCloser(bytes.NewReader(registriesBody)),
		Size: int64(len(registriesBody)),
	}

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
	_, err = cli.ContainerCommit(context.TODO(), volContainerID, dockertypes.ContainerCommitOptions{Reference: regDataImgName})
	if err != nil {
		return nil, err
	}

	// Save all images for the registry
	rdr, err := cli.ImageSave(context.TODO(), append(requiredRegistryImages, regDataImgName))
	if err != nil {
		return nil, err
	}

	// Create artifact for registry images
	registryArtifact, err := util.ArtifactFromReader(types.ArtifactImages, "private-registry.tar", rdr)
	if err != nil {
		return nil, err
	}

	return []*types.Artifact{caCertificate, registriesYamlArtifact, deploymentManifest, registryArtifact}, nil
}
