package images

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/tinyzimmer/k3p/pkg/log"
	"github.com/tinyzimmer/k3p/pkg/types"
)

var registryTmpl = template.Must(template.New("").Parse(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: private-registry
  namespace: kube-system
  labels:
    app: private-registry
spec:
  replicas: 1
  selector:
    matchLabels:
      app: private-registry
  template:
    metadata:
      labels:
        app: private-registry
    spec:
      priorityClassName: system-cluster-critical
      volumes:
        - name: registry-data
          emptyDir: {}
      initContainers:
        - name: data-extractor
          image: private-registry-data:latest
          imagePullPolicy: Never
          command: ['tar', '-xvz', '--file=/var/registry-data.tgz', '--directory=/var/lib/registry']
          volumeMounts:
            - name: registry-data
              mountPath: /var/lib/registry
      containers:
        - name: private-registry
          image: registry:2
          imagePullPolicy: Never
          ports:
            - containerPort: 5000
          volumeMounts:
            - name: registry-data
              mountPath: /var/lib/registry
---
apiVersion: v1
kind: Service
metadata:
  name: private-registry
  namespace: kube-system
  labels:
    app: private-registry
spec:
  type: LoadBalancer
  selector:
    app: private-registry
  ports:
    - port: 5000
      protocol: TCP
      targetPort: 5000
`))

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
