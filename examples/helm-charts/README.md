# Helm chart example

This directory contains a config that can be used for a build with helm charts.
First you must pull the helm charts down in to this directory (you can also use chart directories as well):

```sh
# Add the helm chart repositories
$ helm repo add tinyzimmer https://tinyzimmer.github.io/kvdi/deploy/charts
$ helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
$ helm repo update

# Download the packaged charts
$ helm fetch prometheus-community/kube-prometheus-stack
$ helm fetch tinyzimmer/kvdi
```

Then build a package in this directory using the provided config:

```sh
# For the sake of producing a smaller artifact, we'll use the --exclude-images flag
$ k3p build --exclude-images --name kvdi
2020/12/11 20:32:08  [INFO]     Building package "kvdi"
2020/12/11 20:32:08  [INFO]     Detecting latest k3s version for channel stable
2020/12/11 20:32:09  [INFO]     Latest k3s version is v1.19.4+k3s1
2020/12/11 20:32:09  [INFO]     Packaging distribution for version "v1.19.4+k3s1" using "amd64" architecture
2020/12/11 20:32:09  [INFO]     Downloading core k3s components
2020/12/11 20:32:09  [INFO]     Fetching checksums...
2020/12/11 20:32:09  [INFO]     Fetching k3s install script...
2020/12/11 20:32:09  [INFO]     Fetching k3s binary...
2020/12/11 20:32:09  [INFO]     Skipping bundling k3s airgap images with the package
2020/12/11 20:32:09  [INFO]     Validating checksums...
2020/12/11 20:32:09  [INFO]     Searching "/home/tinyzimmer/devel/k3p/examples/kvdi" for kubernetes manifests to include in the archive
2020/12/11 20:32:09  [INFO]     Detected helm chart at /home/tinyzimmer/devel/k3p/examples/kvdi/kube-prometheus-stack-12.8.0.tgz
2020/12/11 20:32:09  [INFO]     Detected helm chart at /home/tinyzimmer/devel/k3p/examples/kvdi/kvdi-v0.1.1.tgz
2020/12/11 20:32:09  [INFO]     Skipping bundling container images with the package
2020/12/11 20:32:09  [INFO]     Writing package metadata
2020/12/11 20:32:09  [INFO]     Archiving version "latest" of "kvdi" to "/home/tinyzimmer/devel/k3p/examples/kvdi/package.tar"
```
