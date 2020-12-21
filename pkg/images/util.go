package images

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

func getDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func filtersForImage(image string) filters.Args {
	return filters.NewArgs(filters.Arg("reference", image))
}

func sanitizeImageNameSlice(images []string) []string {
	out := make([]string, 0)
	for _, img := range images {
		if sani := sanitizeImageName(img); sani != "" {
			out = append(out, sani)
		}
	}
	return out
}

func sanitizeImageName(image string) string {
	imgParts := strings.Split(image, "@")
	if len(imgParts) > 1 {
		image = imgParts[0]
	}
	// The leading docker.io messes with image list
	if strings.HasPrefix(image, "docker.io/") {
		image = strings.TrimPrefix(image, "docker.io/")
	}
	// Extra check that no empty strings made it in - this check should probably be done somewhere else
	return strings.TrimSpace(image)
}

func ensureImagePulled(cli *client.Client, image, arch string, pullPolicy types.PullPolicy) error {
	switch pullPolicy {
	case types.PullPolicyNever:
		imgs, err := cli.ImageList(context.TODO(), dockertypes.ImageListOptions{
			Filters: filtersForImage(image),
		})
		if err != nil {
			return err
		}
		if len(imgs) == 0 {
			return fmt.Errorf("Image %s is not present on the machine", image)
		}
	case types.PullPolicyIfNotPresent:
		log.Debug("Checking local docker images for", image)
		imgs, err := cli.ImageList(context.TODO(), dockertypes.ImageListOptions{
			Filters: filtersForImage(image),
		})
		if err != nil {
			log.Debugf("Error trying to list images for %s: %s\n", image, err.Error())
		}
		if imgs == nil || len(imgs) != 1 {
			return pullImage(cli, image, arch)
		}
		log.Infof("Image %s already present on the machine\n", image)
	case types.PullPolicyAlways:
		return pullImage(cli, image, arch)
	}
	return nil
}

func pullImage(cli *client.Client, image, arch string) error {
	log.Infof("Pulling image for %s\n", image)
	rdr, err := cli.ImagePull(context.TODO(), image, dockertypes.ImagePullOptions{Platform: arch})
	if err != nil {
		return err
	}
	log.LevelReader(log.LevelDebug, rdr)
	return nil
}

func registryContainerConfigs() (*container.Config, *container.HostConfig) {
	// Expose a random local port to the registry
	exposedPorts, portBindings, err := nat.ParsePortSpecs([]string{"0:5000"})
	if err != nil {
		log.Fatal(err)
	}
	containerConig := &container.Config{
		Image:        "registry:2",
		ExposedPorts: exposedPorts,
		Volumes: map[string]struct{}{
			"/var/lib/registry": struct{}{},
		},
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}
	return containerConig, hostConfig
}

func registryVolumeContainerConfigs(regsitryContainerID string) (*container.Config, *container.HostConfig) {
	containerConfig := &container.Config{
		Image: "busybox",
		Volumes: map[string]struct{}{
			"/var/lib/registry": struct{}{},
		},
		Cmd: strslice.StrSlice([]string{
			"tar", "-cvz", "--file=/var/registry-data.tgz", "--directory=/var/lib/registry", ".",
		}), //                                   |- should be constant -|
	}
	hostConfig := &container.HostConfig{
		VolumesFrom: []string{regsitryContainerID},
	}
	return containerConfig, hostConfig
}

func createAndStartContainer(cli *client.Client, containerConfig *container.Config, hostConfig *container.HostConfig) (id string, err error) {
	cont, err := cli.ContainerCreate(context.TODO(), containerConfig, hostConfig, nil, "")
	if err != nil {
		return "", err
	}
	if err := cli.ContainerStart(context.TODO(), cont.ID, dockertypes.ContainerStartOptions{}); err != nil {
		defer func() {
			if cerr := cli.ContainerRemove(context.TODO(), cont.ID, dockertypes.ContainerRemoveOptions{
				Force:         true,
				RemoveVolumes: true,
			}); cerr != nil {
				log.Warning("Error removing failed container:", cerr)
			}
		}()
		return "", err
	}
	return cont.ID, nil
}

func getHostPortForContainer(cli *client.Client, containerID string, portProto string) (string, error) {
	deets, err := cli.ContainerInspect(context.TODO(), containerID)
	if err != nil {
		return "", err
	}
	localPortMap, ok := deets.NetworkSettings.Ports["5000/tcp"]
	if !ok {
		return "", fmt.Errorf("Could not determine host port for %s on %s from %+v", portProto, containerID, deets.HostConfig.PortBindings)
	}
	localPort := localPortMap[0].HostPort
	return localPort, nil
}

func generateCACertificate(name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate a 4096-bit RSA private key
	caPriv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	fixName := strings.Replace(name, "_", "-", -1)
	caCert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("%s-registry-ca", fixName),
			Organization: []string{fmt.Sprintf("%s-private-registry", fixName)},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10), // 10 years - obviously needs to be handled better
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDerBytes, err := x509.CreateCertificate(rand.Reader, caCert, caCert, caPriv.Public(), caPriv)
	if err != nil {
		return nil, nil, err
	}
	caCertSigned, err := x509.ParseCertificate(caDerBytes)
	if err != nil {
		return nil, nil, err
	}
	return caCertSigned, caPriv, nil
}

func generateRegistryCertificate(caCert *x509.Certificate, caKey *rsa.PrivateKey, name string) (*x509.Certificate, *rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	fixName := strings.Replace(name, "_", "-", -1)
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   fmt.Sprintf("%s-private-registry", fixName),
			Organization: []string{fmt.Sprintf("%s-private-registry", fixName)},
		},
		DNSNames:              []string{"localhost", "kubenab.kube-system.svc"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10), // 10 years - obviously needs to be handled better
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, cert, caCert, priv.Public(), caKey)
	if err != nil {
		return nil, nil, err
	}
	certSigned, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, nil, err
	}
	return certSigned, priv, nil
}

func encodeToPEM(rawCert *x509.Certificate, rawKey *rsa.PrivateKey) (cert, key []byte, err error) {
	var certout bytes.Buffer

	// encode the certificate
	if err := pem.Encode(&certout, &pem.Block{Type: "CERTIFICATE", Bytes: rawCert.Raw}); err != nil {
		return nil, nil, err
	}
	certBytes := certout.Bytes()

	var keyout bytes.Buffer

	// encode the private key
	if err := pem.Encode(&keyout, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rawKey)}); err != nil {
		return nil, nil, err
	}
	keyBytes := keyout.Bytes()

	return certBytes, keyBytes, nil
}

func waitForLocalRegistry(port string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client := &http.Client{Timeout: time.Second * 2}
	for {
		select {
		case <-ctx.Done():
			return errors.New("Time out reached waiting for registry to be ready")
		default:
			res, err := client.Get(fmt.Sprintf("http://localhost:%s/v2/_catalog", port))
			if err != nil {
				log.Debug("Error waiting for registry to be ready, will retry:", err)
				continue
			}
			if res.StatusCode != http.StatusOK {
				log.Debug("Non-200 status code from registry catalog, will retry:", err)
				continue
			}
			log.Debug("Local registry is ready")
			return nil
		}
	}
}
