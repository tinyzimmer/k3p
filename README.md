# k3p

A `k3s` packager and installer, primarily intended for airgapped deployments

## TODO

Next up is packaging the actual kubernetes manifests. A parser that finds `yaml` files that have
valid kubernetes objects, and adds them to the bundle. The base interface needs to be able to be
extended to allow for supercedence. For example, if a `helm` parser were to find a chart, it needs to be able
to invalidate that directory for the raw parser (so the user can have a mix, but the raw parser won't try to read templates).

## Examples

```bash
[tinyzimmer@base k3p]$ make
cd cmd/k3p && \
        go build -o ../../dist/k3p .
```


```bash
[tinyzimmer@base k3p]$ dist/k3p help build
Build an embedded k3s distribution package

Usage:
  k3p build [flags]

Flags:
  -a, --arch string        The architecture to package the distribution for (default "amd64")
  -e, --exclude strings    Directories to exclude when reading the manifest directory
  -h, --help               help for build
  -m, --manifests string   The directory to scan for kubernetes manifests, defaults to the current directory (default "/home/tinyzimmer/devel/k3p")
  -o, --output string      The file to save the distribution package to (default "/home/tinyzimmer/devel/k3p/package.tar")
  -V, --version string     The k3s version to bundle with the package (default "latest")

Global Flags:
  -v, --verbose   Enable verbose logging
```

```bash
[tinyzimmer@base k3p]$ dist/k3p build
2020/11/26 18:26:08 Detecting latest k3s version
2020/11/26 18:26:09 Latest k3s version is v1.19.4+k3s1
2020/11/26 18:26:09 Using temporary build directory: /tmp/016041342
2020/11/26 18:26:09 Packaging distribution for version "v1.19.4+k3s1" using "amd64" architecture
2020/11/26 18:26:09 Downloading core k3s components
2020/11/26 18:26:09 Fetching checksums...
2020/11/26 18:26:10 Fetching k3s install script...
2020/11/26 18:26:11 Fetching k3s binary...
2020/11/26 18:26:38 Fetching k3s airgap images...
2020/11/26 18:30:27 Validating checksums...
2020/11/26 18:30:28 Parsing kubernetes manifests for container images to download
2020/11/26 18:30:28 Found Deployment: cert-manager-webhook
2020/11/26 18:30:28 Found Deployment: cert-manager
2020/11/26 18:30:28 Found Job: cert-manager-webhook-ca-sync
2020/11/26 18:30:28 Found CronJob: cert-manager-webhook-ca-sync
2020/11/26 18:30:28 Detected the following images to bundle with the package: [quay.io/jetstack/cert-manager-webhook:v0.6.2 quay.io/jetstack/cert-manager-controller:v0.6.2 quay.io/munnerz/apiextensions-ca-helper:v0.1.0]
```
