# k3p

A `k3s` packager and installer, primarily intended for airgapped deployments

For documentation on `k3p` usage, see the [command docs here](doc/k3p.md).

## Makefile

The following commands are avaiable in the makefile.

```bash
make [build]               # Builds k3p to dist/k3p
make pkg [PKG_ARGS="..."]  # Builds a k3s package using k3p with optional arguments
make lint                  # Lints the codebase

# If you have local nodes to play with, you can set NODE_USER and NODES in your environment
# and use the following:
#
# Example
# 
# $ export NODES="192.168.1.100 192.168.1.101 192.168.1.102"
# $ export NODE_USER=root
# 
# Then use the following targets

make dist-node-1     # Install k3p and copy the package built above to the first node in NODES
make node-shell-2    # Get a bash shell on the second node in NODES
make clean-server-3  # Uninstall the k3s server from node 3
make clean-agent-3   # Uninstall the k3s agent from node 3

# Same as above but runs against all nodes in NODES
make dist-node-all clean-server-all clean-agent-all
```

## Examples

### Standalone

```bash
# Create the package on any machine
[tinyzimmer@base k3p-wsl]$ dist/k3p build
[INFO] 2020/12/02 07:13:05 Detecting latest k3s version
[INFO] 2020/12/02 07:13:06 Latest k3s version is v1.19.4+k3s1
[INFO] 2020/12/02 07:13:06 Packaging distribution for version "v1.19.4+k3s1" using "amd64" architecture
[INFO] 2020/12/02 07:13:06 Downloading core k3s components
[INFO] 2020/12/02 07:13:06 Fetching checksums...
[INFO] 2020/12/02 07:13:07 Fetching k3s install script...
[INFO] 2020/12/02 07:13:07 Fetching k3s binary...
[INFO] 2020/12/02 07:13:19 Fetching k3s airgap images...
[INFO] 2020/12/02 07:15:41 Validating checksums...
[INFO] 2020/12/02 07:15:45 Searching for kubernetes manifests to include in the archive
[INFO] 2020/12/02 07:15:45 Packaging helm chart: "/home/tinyzimmer/devel/k3p/example-manifests/my-chart"
[INFO] 2020/12/02 07:15:46 Detected kubernetes manifest: "/home/tinyzimmer/devel/k3p/example-manifests/whoami.yaml"
[WARNING] 2020/12/02 07:15:46 Skipping "/home/tinyzimmer/devel/k3p/hack/coreos-config.yaml" since it contains invalid kubernetes yaml
[INFO] 2020/12/02 07:15:46 Parsing kubernetes manifests for container images to download
[INFO] 2020/12/02 07:15:46 Detected helm chart at /home/tinyzimmer/devel/k3p/example-manifests/my-chart
[INFO] 2020/12/02 07:15:46 Found appsv1 Deployment: RELEASE-NAME-my-chart
[INFO] 2020/12/02 07:15:46 Found Pod: RELEASE-NAME-my-chart-test-connection
[INFO] 2020/12/02 07:15:46 Found appsv1 Deployment: whoami
[INFO] 2020/12/02 07:15:46 Detected the following images to bundle with the package: [nginx:1.16.0 busybox traefik/whoami:latest]
[INFO] 2020/12/02 07:15:46 Pulling image for nginx:1.16.0
[INFO] 2020/12/02 07:15:48 Pulling image for busybox
[INFO] 2020/12/02 07:15:50 Pulling image for traefik/whoami:latest
[INFO] 2020/12/02 07:15:55 Archiving bundle to "/home/tinyzimmer/devel/k3p/package.tar"

# Copy it to the location (works in Linux, Mac, or WSL2)

# Install the package
[core@coreos1 ~]$ sudo ./k3p install package.tar 
[INFO] 2020/12/02 05:18:48 Extracting "package.tar"
[INFO] 2020/12/02 05:18:48 Installing binaries to /usr/local/bin/
[INFO] 2020/12/02 05:18:48 Installing scripts to /usr/local/bin/k3p
[INFO] 2020/12/02 05:18:48 Installing images to /var/lib/rancher/k3s/agent/images
[INFO] 2020/12/02 05:18:49 Installing kubernetes manifests to /var/lib/rancher/k3s/server/manifests
[INFO] 2020/12/02 05:18:49 Running k3s installation script
[INFO]  Skipping k3s download and verify
[INFO]  Skipping installation of SELinux RPM
[INFO]  Creating /usr/local/bin/kubectl symlink to k3s
[INFO]  Creating /usr/local/bin/crictl symlink to k3s
[INFO]  Skipping /usr/local/bin/ctr symlink to k3s, command exists in PATH at /usr/bin/ctr
[INFO]  Creating killall script /usr/local/bin/k3s-killall.sh
[INFO]  Creating uninstall script /usr/local/bin/k3s-uninstall.sh
[INFO]  env: Creating environment file /etc/systemd/system/k3s.service.env
[INFO]  systemd: Creating service file /etc/systemd/system/k3s.service
[INFO]  systemd: Enabling k3s unit
Created symlink /etc/systemd/system/multi-user.target.wants/k3s.service → /etc/systemd/system/k3s.service.
[INFO]  systemd: Starting k3s
[INFO] 2020/12/02 05:18:57 The cluster has been installed. For additional details run `kubectl cluster-info`.

[core@coreos1 ~]$ sudo kubectl cluster-info
Kubernetes master is running at https://127.0.0.1:6443
CoreDNS is running at https://127.0.0.1:6443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
Metrics-server is running at https://127.0.0.1:6443/api/v1/namespaces/kube-system/services/https:metrics-server:/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.

[core@coreos1 ~]$ curl -H "Host: whoami.local" localhost
Hostname: whoami-5dc4dd9cdf-gvcld
IP: 127.0.0.1
IP: ::1
IP: 10.42.0.2
IP: fe80::e4c2:2eff:fecd:49ad
RemoteAddr: 10.42.0.9:35524
GET / HTTP/1.1
Host: whoami.local
User-Agent: curl/7.69.1
Accept: */*
Accept-Encoding: gzip
X-Forwarded-For: 10.42.0.10
X-Forwarded-Host: whoami.local
X-Forwarded-Port: 80
X-Forwarded-Proto: http
X-Forwarded-Server: traefik-5dd496474-dnnhv
X-Real-Ip: 10.42.0.10

# Uninstalls k3s and all its assets
[core@coreos1 ~]$ k3s-uninstall.sh 
```

### HA

```bash
# On the initial HA server
[core@coreos1 ~]$ sudo ./k3p install package.tar --init-ha
[INFO] 2020/12/02 05:22:01 Extracting "package.tar"
[INFO] 2020/12/02 05:22:02 Installing binaries to /usr/local/bin/
[INFO] 2020/12/02 05:22:02 Installing scripts to /usr/local/bin/k3p
[INFO] 2020/12/02 05:22:02 Installing images to /var/lib/rancher/k3s/agent/images
[INFO] 2020/12/02 05:22:02 Installing kubernetes manifests to /var/lib/rancher/k3s/server/manifests
[INFO] 2020/12/02 05:22:02 Generating a node token for additional control-plane instances
[INFO] 2020/12/02 05:22:02 You can join new servers to the control-plane with the following token: SanBSUOSD3CgspP6DHkyVcmh7VmyvbecXYInTij6utaljynHhERqrqSMw3CqN0x8IH6boKus49lKYcaEvqoVBlRIyqQ7NUNx5YQxYEmiIECgR69fkIC2MsHR9BHwuh8MmIXfSaJLq7PqdOiKpZrWfEHbCSt63Ctloeqip7oU5bR3L2ygANwmXW2fUKRp0DFhmP697IFfH6zoKYWxeGiHQ3JztrNh63thEYSFWIusu92IKFH0DVVqNEZYpKWsjkS4
[INFO] 2020/12/02 05:22:02 Applying extra k3s arguments: " --cluster-init"
[INFO] 2020/12/02 05:22:02 Running k3s installation script
[INFO]  Skipping k3s download and verify
[INFO]  Skipping installation of SELinux RPM
[INFO]  Creating /usr/local/bin/kubectl symlink to k3s
[INFO]  Creating /usr/local/bin/crictl symlink to k3s
[INFO]  Skipping /usr/local/bin/ctr symlink to k3s, command exists in PATH at /usr/bin/ctr
[INFO]  Creating killall script /usr/local/bin/k3s-killall.sh
[INFO]  Creating uninstall script /usr/local/bin/k3s-uninstall.sh
[INFO]  env: Creating environment file /etc/systemd/system/k3s.service.env
[INFO]  systemd: Creating service file /etc/systemd/system/k3s.service
[INFO]  systemd: Enabling k3s unit
Created symlink /etc/systemd/system/multi-user.target.wants/k3s.service → /etc/systemd/system/k3s.service.
[INFO]  systemd: Starting k3s
[INFO] 2020/12/02 05:22:12 The cluster has been installed. For additional details run `kubectl cluster-info`.

[core@coreos1 ~]$ sudo ./k3p token get server
SanBSUOSD3CgspP6DHkyVcmh7VmyvbecXYInTij6utaljynHhERqrqSMw3CqN0x8IH6boKus49lKYcaEvqoVBlRIyqQ7NUNx5YQxYEmiIECgR69fkIC2MsHR9BHwuh8MmIXfSaJLq7PqdOiKpZrWfEHbCSt63Ctloeqip7oU5bR3L2ygANwmXW2fUKRp0DFhmP697IFfH6zoKYWxeGiHQ3JztrNh63thEYSFWIusu92IKFH0DVVqNEZYpKWsjkS4

# On a second AND third node
## --join-role agent is used for adding worker nodes (in standalone also) and utilizes a different 
## token that can be retrieved with "k3p token get agent"

[core@coreos2 ~]$ sudo ./k3p install package.tar --join https://172.17.113.136:6443 --join-role server --token SanBSUOSD3CgspP6DHkyVcmh7VmyvbecXYInTij6utaljynHhERqrqSMw3CqN0x8IH6boKus49lKYcaEvqoVBlRIyqQ7NUNx5YQxYEmiIECgR69fkIC2MsHR9BHwuh8MmIXfSaJLq7PqdOiKpZrWfEHbCSt63Ctloeqip7oU5bR3L2ygANwmXW2fUKRp0DFhmP697IFfH6zoKYWxeGiHQ3JztrNh63thEYSFWIusu92IKFH0DVVqNEZYpKWsjkS4
[INFO] 2020/12/02 05:24:33 Extracting "package.tar"
[INFO] 2020/12/02 05:24:34 Installing binaries to /usr/local/bin/
[INFO] 2020/12/02 05:24:34 Installing scripts to /usr/local/bin/k3p
[INFO] 2020/12/02 05:24:34 Installing images to /var/lib/rancher/k3s/agent/images
[INFO] 2020/12/02 05:24:35 Installing kubernetes manifests to /var/lib/rancher/k3s/server/manifests
[INFO] 2020/12/02 05:24:35 Joining server at: https://172.17.113.136:6443
[INFO] 2020/12/02 05:24:35 Running k3s installation script
[INFO]  Skipping k3s download and verify
[INFO]  Skipping installation of SELinux RPM
[INFO]  Creating /usr/local/bin/kubectl symlink to k3s
[INFO]  Creating /usr/local/bin/crictl symlink to k3s
[INFO]  Skipping /usr/local/bin/ctr symlink to k3s, command exists in PATH at /usr/bin/ctr
[INFO]  Creating killall script /usr/local/bin/k3s-killall.sh
[INFO]  Creating uninstall script /usr/local/bin/k3s-uninstall.sh
[INFO]  env: Creating environment file /etc/systemd/system/k3s.service.env
[INFO]  systemd: Creating service file /etc/systemd/system/k3s.service
[INFO]  systemd: Enabling k3s unit
Created symlink /etc/systemd/system/multi-user.target.wants/k3s.service → /etc/systemd/system/k3s.service.
[INFO]  systemd: Starting k3s
[INFO] 2020/12/02 05:24:58 The cluster has been installed. For additional details run `kubectl cluster-info`.

[core@coreos3 ~]$ sudo ./k3p install package.tar --join https://172.17.113.136:6443 --join-role server --token SanBSUOSD3CgspP6DHkyVcmh7VmyvbecXYInTij6utaljynHhERqrqSMw3CqN0x8IH6boKus49lKYcaEvqoVBlRIyqQ7NUNx5YQxYEmiIECgR69fkIC2MsHR9BHwuh8MmIXfSaJLq7PqdOiKpZrWfEHbCSt63Ctloeqip7oU5bR3L2ygANwmXW2fUKRp0DFhmP697IFfH6zoKYWxeGiHQ3JztrNh63thEYSFWIusu92IKFH0DVVqNEZYpKWsjkS4
[INFO] 2020/12/02 05:24:57 Extracting "package.tar"
[INFO] 2020/12/02 05:24:57 Installing binaries to /usr/local/bin/
[INFO] 2020/12/02 05:24:57 Installing scripts to /usr/local/bin/k3p
[INFO] 2020/12/02 05:24:57 Installing images to /var/lib/rancher/k3s/agent/images
[INFO] 2020/12/02 05:24:59 Installing kubernetes manifests to /var/lib/rancher/k3s/server/manifests
[INFO] 2020/12/02 05:24:59 Joining server at: https://172.17.113.136:6443
[INFO] 2020/12/02 05:24:59 Running k3s installation script
[INFO]  Skipping k3s download and verify
[INFO]  Skipping installation of SELinux RPM
[INFO]  Creating /usr/local/bin/kubectl symlink to k3s
[INFO]  Creating /usr/local/bin/crictl symlink to k3s
[INFO]  Skipping /usr/local/bin/ctr symlink to k3s, command exists in PATH at /usr/bin/ctr
[INFO]  Creating killall script /usr/local/bin/k3s-killall.sh
[INFO]  Creating uninstall script /usr/local/bin/k3s-uninstall.sh
[INFO]  env: Creating environment file /etc/systemd/system/k3s.service.env
[INFO]  systemd: Creating service file /etc/systemd/system/k3s.service
[INFO]  systemd: Enabling k3s unit
Created symlink /etc/systemd/system/multi-user.target.wants/k3s.service → /etc/systemd/system/k3s.service.
[INFO]  systemd: Starting k3s
[INFO] 2020/12/02 05:25:13 The cluster has been installed. For additional details run `kubectl cluster-info`.

# Back on the master node. You can retrieve the kubeconfig for use elsewhere at /etc/rancher/k3s/k3s.yaml.
[core@coreos1 ~]$ sudo kubectl get node -o wide
NAME      STATUS   ROLES         AGE     VERSION        INTERNAL-IP      EXTERNAL-IP   OS-IMAGE                        KERNEL-VERSION           CONTAINER-RUNTIME
coreos1   Ready    etcd,master   3m51s   v1.19.4+k3s1   172.17.113.136   <none>        Fedora CoreOS 32.20201104.3.0   5.8.17-200.fc32.x86_64   containerd://1.4.1-k3s1
coreos2   Ready    etcd,master   62s     v1.19.4+k3s1   172.17.113.137   <none>        Fedora CoreOS 32.20201104.3.0   5.8.17-200.fc32.x86_64   containerd://1.4.1-k3s1
coreos3   Ready    etcd,master   48s     v1.19.4+k3s1   172.17.113.130   <none>        Fedora CoreOS 32.20201104.3.0   5.8.17-200.fc32.x86_64   containerd://1.4.1-k3s1
```

### Join new nodes remotely over SSH :wink:

```bash
[core@coreos1 ~]$ sudo ./k3p node add --help
Add a new node to the cluster

Usage:
  k3p node add NODE [flags]

Flags:
  -h, --help                 help for add
  -r, --node-role string     Whether to join the instance as a 'server' or 'agent' (default "agent")
  -k, --private-key string   A private key to use for SSH authentication, if not provided you will be prompted for a password
  -p, --ssh-port int         The port to use when connecting to the remote instance over SSH (default 22)
  -u, --ssh-user string      The remote user to use for SSH authentication (default "root")

Global Flags:
      --cache-dir string   Override the default location for cached k3s assets (default "/root/.k3p/cache")
      --tmp-dir string     Override the default tmp directory (default "/tmp")
  -v, --verbose            Enable verbose logging

[core@coreos1 ~]$ sudo ./k3p node add 172.17.113.137 -v --ssh-user=core --private-key=/var/home/core/.ssh/id_rsa 

[DEBUG] 2020/12/02 19:10:18 Default cache dir is "/root/.k3p/cache"
[INFO] 2020/12/02 19:10:18 Determining current k3s external listening address
[DEBUG] 2020/12/02 19:10:18 Scanning "/proc/83797/net/tcp" for remote port
[DEBUG] 2020/12/02 19:10:18 K3s is listening on 172.17.113.136
[INFO] 2020/12/02 19:10:18 Connecting to server 172.17.113.137 on port 22
[DEBUG] 2020/12/02 19:10:18 Using SSH user: core
[DEBUG] 2020/12/02 19:10:18 Using SSH pubkey authentication
[DEBUG] 2020/12/02 19:10:18 Loading SSH key from "/var/home/core/.ssh/id_rsa"
[DEBUG] 2020/12/02 19:10:18 Creating SSH connection with 172.17.113.137:22 over TCP
[INFO] 2020/12/02 19:10:18 Copying package manifest to the new node
[DEBUG] 2020/12/02 19:10:18 Executing command on remote: mkdir -p /var/lib/rancher/k3s/server
[DEBUG] 2020/12/02 19:10:18 Executing command on remote: sudo tee /var/lib/rancher/k3s/server/package.tar
[INFO] 2020/12/02 19:10:21 Copying the k3p binary to the new node
[DEBUG] 2020/12/02 19:10:21 Executing command on remote: mkdir -p /var/home/core
[DEBUG] 2020/12/02 19:10:21 Executing command on remote: sudo tee /var/home/core/k3p
[DEBUG] 2020/12/02 19:10:21 Reading agent join token from /var/lib/rancher/k3s/server/node-token
[INFO] 2020/12/02 19:10:21 Joining new server instance at 172.17.113.137
[DEBUG] 2020/12/02 19:10:21 Executing command on remote: sudo /var/home/core/k3p install /var/lib/rancher/k3s/server/package.tar --join https://172.17.113.136:6443 --join-role agent --join-token <redacted>
[INFO]  Skipping k3s download and verify
[INFO]  Skipping installation of SELinux RPM
[INFO]  Skipping /usr/local/bin/kubectl symlink to k3s, already exists
[INFO]  Skipping /usr/local/bin/crictl symlink to k3s, already exists
[INFO]  Skipping /usr/local/bin/ctr symlink to k3s, command exists in PATH at /usr/bin/ctr
[INFO]  Creating killall script /usr/local/bin/k3s-killall.sh
[INFO]  Creating uninstall script /usr/local/bin/k3s-agent-uninstall.sh
[INFO]  env: Creating environment file /etc/systemd/system/k3s-agent.service.env
[INFO]  systemd: Creating service file /etc/systemd/system/k3s-agent.service
[INFO]  systemd: Enabling k3s-agent unit
[INFO]  systemd: Starting k3s-agent

[core@coreos1 ~]$ sudo kubectl get nodes
NAME      STATUS   ROLES    AGE   VERSION
coreos1   Ready    master   11m   v1.19.4+k3s1
coreos2   Ready    <none>   59s   v1.19.4+k3s1
```