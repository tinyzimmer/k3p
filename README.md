# k3p

A `k3s` packager and installer, primarily intended for airgapped deployments

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

### E2E

```bash
[tinyzimmer@base k3p-wsl]$ dist/k3p build
INFO: 2020/11/28 12:06:20 Detecting latest k3s version
INFO: 2020/11/28 12:06:20 Latest k3s version is v1.19.4+k3s1
INFO: 2020/11/28 12:06:20 Packaging distribution for version "v1.19.4+k3s1" using "amd64" architecture
INFO: 2020/11/28 12:06:20 Downloading core k3s components
INFO: 2020/11/28 12:06:20 Fetching checksums...
INFO: 2020/11/28 12:06:20 Fetching k3s install script...
INFO: 2020/11/28 12:06:20 Fetching k3s binary...
INFO: 2020/11/28 12:06:20 Fetching k3s airgap images...
INFO: 2020/11/28 12:06:21 Validating checksums...
INFO: 2020/11/28 12:06:22 Searching for kubernetes manifests to include in the archive
INFO: 2020/11/28 12:06:22 Detected kubernetes manifest: "/home/aizimmerman/devel/k3p/example/whoami.yaml"
INFO: 2020/11/28 12:06:22 Parsing kubernetes manifests for container images to download
INFO: 2020/11/28 12:06:22 Found Deployment: whoami
INFO: 2020/11/28 12:06:22 Detected the following images to bundle with the package: [traefik/whoami:latest]
INFO: 2020/11/28 12:06:22 Pulling image for traefik/whoami:latest
INFO: 2020/11/28 12:06:24 Archiving bundle to "/home/aizimmerman/devel/k3p/package.tar"

[tinyzimmer@base k3p-wsl]$ sudo dist/k3p install package.tar 
INFO: 2020/11/28 12:06:56 Extracting "package.tar"
INFO: 2020/11/28 12:06:56 Installing binaries to /usr/local/bin/
INFO: 2020/11/28 12:06:56 Installing scripts to /usr/local/bin/k3p
INFO: 2020/11/28 12:06:56 Installing images to /var/lib/rancher/k3s/agent/images
INFO: 2020/11/28 12:06:56 Install kubernetes manifests to /var/lib/rancher/k3s/server/manifests
INFO: 2020/11/28 12:06:56 Running k3s installation script
[INFO]  Skipping k3s download and verify
[INFO]  Skipping installation of SELinux RPM
[INFO]  Skipping /usr/local/bin/kubectl symlink to k3s, command exists in PATH at /usr/bin/kubectl
[INFO]  Creating /usr/local/bin/crictl symlink to k3s
[INFO]  Creating /usr/local/bin/ctr symlink to k3s
[INFO]  Creating killall script /usr/local/bin/k3s-killall.sh
[INFO]  Creating uninstall script /usr/local/bin/k3s-uninstall.sh
[INFO]  env: Creating environment file /etc/systemd/system/k3s.service.env
[INFO]  systemd: Creating service file /etc/systemd/system/k3s.service
[INFO]  systemd: Enabling k3s unit
Created symlink /etc/systemd/system/multi-user.target.wants/k3s.service → /etc/systemd/system/k3s.service.
[INFO]  systemd: Starting k3s
Job for k3s.service failed because the control process exited with error code.
See "systemctl status k3s.service" and "journalctl -xe" for details.
ERROR: 2020/11/28 12:06:57 exit status 1

# K3s install script fails on WSL but install actually works inside a genie bottle anyway.
# Need to setup an environment to see if works correctly in Linux. Virtualbox having issues
# on latest Win10, and I nuked my old dual boot environment.

[tinyzimmer@base k3p-wsl]$ curl -H "Host: whoami.local" localhost
Hostname: whoami-5db874f58d-xcpgd
IP: 127.0.0.1
IP: ::1
IP: 10.42.0.6
IP: fe80::84be:b7ff:feaf:8ce6
RemoteAddr: 10.42.0.5:34002
GET / HTTP/1.1
Host: whoami.local
User-Agent: curl/7.73.0
Accept: */*
Accept-Encoding: gzip
X-Forwarded-For: 10.42.0.4
X-Forwarded-Host: whoami.local
X-Forwarded-Port: 80
X-Forwarded-Proto: http
X-Forwarded-Server: traefik-5dd496474-b4x7m
X-Real-Ip: 10.42.0.4
```
