# k3p

A `k3s` packager and installer, primarily intended for airgapped deployments

For documentation on `k3p` usage, see the [command docs here](doc/k3p.md).

## TODO:

- API Port is hard coded in a lot of places, need to make `k3p install --api-port=` available across the board.
- Per node configs

## Quickstart

Will publish releases via actions in the future. For now, on a system with `git` and `go` installed.

```bash
git clone https://github.com/tinyzimmer/k3p
cd k3p
make install

# If you do not have make installed, you can build and install manually with:
cd cmd/k3p
go build -o $(go env GOPATH)/bin/k3p .
```

You can build a package with the `build` command. By default it will scan your current directory, and
detect objects to be included in the archive. See the usage documentation for other configuration options.

```bash
$ k3p build 
2020/12/10 10:49:37  [INFO]     Generated name for package "intelligent_wu"
2020/12/10 10:49:37  [INFO]     Detecting latest k3s version for channel stable
2020/12/10 10:49:38  [INFO]     Latest k3s version is v1.19.4+k3s1
2020/12/10 10:49:38  [INFO]     Packaging distribution for version "v1.19.4+k3s1" using "amd64" architecture
2020/12/10 10:49:38  [INFO]     Downloading core k3s components
2020/12/10 10:49:38  [INFO]     Fetching checksums...
2020/12/10 10:49:38  [INFO]     Fetching k3s install script...
2020/12/10 10:49:38  [INFO]     Fetching k3s binary...
2020/12/10 10:49:38  [INFO]     Fetching k3s airgap images...
2020/12/10 10:49:38  [INFO]     Validating checksums...
2020/12/10 10:49:40  [INFO]     Searching for kubernetes manifests to include in the archive
2020/12/10 10:49:40  [INFO]     Detected kubernetes manifest: "/home/tinyzimmer/devel/k3p/example-manifests/whoami.yaml"
2020/12/10 10:49:40  [INFO]     Parsing kubernetes manifests for container images to download
2020/12/10 10:49:40  [INFO]     Found appsv1 Deployment: whoami
2020/12/10 10:49:40  [INFO]     Detected the following images to bundle with the package: [traefik/whoami:latest]
2020/12/10 10:49:40  [INFO]     Pulling image for traefik/whoami:latest
2020/12/10 10:49:42  [INFO]     Adding container images to package
2020/12/10 10:49:42  [INFO]     Writing package metadata
2020/12/10 10:49:42  [INFO]     Archiving version "latest" of "intelligent_wu" to "/home/tinyzimmer/devel/k3p/package.tar"
```

You can optionally exclude images for the archive, however this is not the default behavior as this project was originally intended
for fully airgapped deployments. Any raw kubernetes `yaml` or `helm` charts found (that are not excluded) will be included and applied
automatically upon installation.

You can then install the package to a system using the `install` command. Installations can be performed either on the local system (requires root),
over a remote SSH connection (requires SSH user have passwordless `sudo`), or to docker containers on the local system similar to [`k3d`](https://github.com/rancher/k3d).

Again, see the usage documentation for more configuration options, and how to use the various installation modes, but to just install to the local system:

```bash
# Will work on any linux system (and WSL2 using genie)
$ sudo k3p install package.tar 
2020/12/10 10:57:56  [INFO]     Loading the archive
2020/12/10 10:57:57  [INFO]     Copying the archive to the rancher installation directory
2020/12/10 10:57:58  [INFO]     Installing binaries to /usr/local/bin
2020/12/10 10:57:58  [INFO]     Installing scripts to /usr/local/bin/k3p-scripts
2020/12/10 10:57:58  [INFO]     Installing images to /var/lib/rancher/k3s/agent/images
2020/12/10 10:57:59  [INFO]     Installing manifests to /var/lib/rancher/k3s/server/manifests
2020/12/10 10:57:59  [INFO]     Running k3s installation script
2020/12/10 10:57:59  [K3S]      [INFO]  Skipping k3s download and verify
2020/12/10 10:57:59  [K3S]      [INFO]  Skipping installation of SELinux RPM
2020/12/10 10:57:59  [K3S]      [INFO]  Skipping /usr/local/bin/kubectl symlink to k3s, command exists in PATH at /usr/bin/kubectl
2020/12/10 10:57:59  [K3S]      [INFO]  Creating /usr/local/bin/crictl symlink to k3s
2020/12/10 10:57:59  [K3S]      [INFO]  Creating /usr/local/bin/ctr symlink to k3s
2020/12/10 10:57:59  [K3S]      [INFO]  Creating killall script /usr/local/bin/k3s-killall.sh
2020/12/10 10:57:59  [K3S]      [INFO]  Creating uninstall script /usr/local/bin/k3s-uninstall.sh
2020/12/10 10:57:59  [K3S]      [INFO]  env: Creating environment file /etc/systemd/system/k3s.service.env
2020/12/10 10:57:59  [K3S]      [INFO]  systemd: Creating service file /etc/systemd/system/k3s.service
2020/12/10 10:57:59  [K3S]      [INFO]  systemd: Enabling k3s unit
2020/12/10 10:57:59  [K3S]      Created symlink /etc/systemd/system/multi-user.target.wants/k3s.service → /etc/systemd/system/k3s.service.
2020/12/10 10:57:59  [K3S]      [INFO]  systemd: Starting k3s
2020/12/10 10:58:06  [INFO]     The cluster has been installed
2020/12/10 10:58:06  [INFO]     You can view the cluster by running `k3s kubectl cluster-info`

$ sudo k3s kubectl cluster-info
Kubernetes master is running at https://127.0.0.1:6443
CoreDNS is running at https://127.0.0.1:6443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
Metrics-server is running at https://127.0.0.1:6443/api/v1/namespaces/kube-system/services/https:metrics-server:/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.

$ sudo k3s kubectl get pod
NAME                      READY   STATUS    RESTARTS   AGE
whoami-5dc4dd9cdf-qvvnz   1/1     Running   0          32s
```

For further information on adding worker nodes and/or setting up HA, you can view the command documentation, 
however more complete documentation will come in the future in the form of [examples](examples/).