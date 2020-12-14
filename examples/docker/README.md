# Playing with Docker

This directory goes into more detail on smoke testing packages with docker. 
If you have used [`k3d`](https://github.com/rancher/k3d) in the past most of this will be familiar to you.

To start off, build the package in this directory (for the purpose of these examples we'll exclude images from the archive):

```bash
# Build the package and give it a unique name
$ k3p build --exclude-images --name=k3p-docker

2020/12/14 10:09:59  [INFO]     Building package "k3p-docker"
2020/12/14 10:09:59  [INFO]     Detecting latest k3s version for channel stable
2020/12/14 10:10:00  [INFO]     Latest k3s version is v1.19.4+k3s1
2020/12/14 10:10:00  [INFO]     Packaging distribution for version "v1.19.4+k3s1" using "amd64" architecture
2020/12/14 10:10:00  [INFO]     Downloading core k3s components
2020/12/14 10:10:00  [INFO]     Fetching checksums...
2020/12/14 10:10:00  [INFO]     Fetching k3s install script...
2020/12/14 10:10:00  [INFO]     Fetching k3s binary...
2020/12/14 10:10:00  [INFO]     Skipping bundling k3s airgap images with the package
2020/12/14 10:10:00  [INFO]     Validating checksums...
2020/12/14 10:10:00  [INFO]     Searching "/home/tinyzimmer/devel/k3p/examples/docker" for kubernetes manifests to include in the archive
2020/12/14 10:10:00  [INFO]     Detected kubernetes manifest: "/home/tinyzimmer/devel/k3p/examples/docker/whoami.yaml"
2020/12/14 10:10:00  [INFO]     Skipping bundling container images with the package
2020/12/14 10:10:00  [INFO]     Writing package metadata
2020/12/14 10:10:00  [INFO]     Archiving version "latest" of "k3p-docker" to "/home/tinyzimmer/devel/k3p/examples/docker/package.tar"
```

To install this package to a simple single node cluster running in docker you can do the following:

```bash
# --write-kubeconfig is optional and will extract the kubeconfig once the server is up
# otherwise instructions are printed for fetching it directly from the container
$ k3p install package.tar --docker --write-kubeconfig kubeconfig.yaml
2020/12/14 10:11:13  [INFO]     Loading the archive
2020/12/14 10:11:13  [INFO]     Creating docker network k3p-docker
2020/12/14 10:11:13  [INFO]     Creating docker volume k3p-docker-server-0
2020/12/14 10:11:14  [INFO]     Copying the archive to the rancher installation directory
2020/12/14 10:11:14  [INFO]     Installing binaries to /usr/local/bin
2020/12/14 10:11:14  [INFO]     Installing scripts to /usr/local/bin/k3p-scripts
2020/12/14 10:11:14  [INFO]     Installing manifests to /var/lib/rancher/k3s/server/manifests
2020/12/14 10:11:14  [INFO]     Running k3s installation script
2020/12/14 10:11:14  [INFO]     Starting k3s docker node k3p-docker-server-0
2020/12/14 10:11:15  [INFO]     Starting k3s docker node k3p-docker-serverlb
2020/12/14 10:11:15  [INFO]     Waiting for server to write the admin kubeconfig
2020/12/14 10:11:17  [INFO]     Writing the kubeconfig to "kubeconfig.yaml"
2020/12/14 10:11:17  [INFO]     The cluster has been installed
2020/12/14 10:11:17  [INFO]     You can view the cluster by running `kubectl --kubeconfig kubeconfig.yaml cluster-info`

$ kubectl --kubeconfig kubeconfig.yaml cluster-info
Kubernetes master is running at https://127.0.0.1:6443
CoreDNS is running at https://127.0.0.1:6443/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy
Metrics-server is running at https://127.0.0.1:6443/api/v1/namespaces/kube-system/services/https:metrics-server:/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.

$ kubectl --kubeconfig kubeconfig.yaml get pod
NAME                      READY   STATUS    RESTARTS   AGE
whoami-5db874f58d-dcx48   1/1     Running   0          54s

# To remove the cluster when you are done
$ k3p uninstall --name=k3p-docker   # The --name flag supports tab completion
2020/12/14 10:13:53  [INFO]     Removing docker cluster k3p-docker
2020/12/14 10:13:53  [INFO]     Removing docker container and volumes for k3p-docker-serverlb
2020/12/14 10:13:53  [INFO]     Removing docker container and volumes for k3p-docker-server-0
2020/12/14 10:13:54  [INFO]     Removing docker network k3p-docker
```

You can specify server/agent count and configurations also (this is the same as for a regular install)

```bash
$ k3p install package.tar --docker --write-kubeconfig kubeconfig.yaml \
    --servers 3 --agents 3 \             # Specify number of server and agent nodes
    --k3s-server-arg="--disable=traefik" # can be specified multiple times, there is also an agent equivalent

# ...
# ...

$ kubectl --kubeconfig kubeconfig.yaml get node
NAME                  STATUS   ROLES         AGE   VERSION
k3p-docker-agent-0    Ready    worker        47s   v1.19.4+k3s1
k3p-docker-agent-1    Ready    worker        50s   v1.19.4+k3s1
k3p-docker-agent-2    Ready    worker        49s   v1.19.4+k3s1
k3p-docker-server-0   Ready    etcd,master   55s   v1.19.4+k3s1
k3p-docker-server-1   Ready    etcd,master   24s   v1.19.4+k3s1
k3p-docker-server-2   Ready    etcd,master   44s   v1.19.4+k3s1

$ kubectl --kubeconfig kubeconfig.yaml get pod -A
NAMESPACE     NAME                                     READY   STATUS    RESTARTS   AGE
default       whoami-5db874f58d-xtgrl                  1/1     Running   0          2m42s
kube-system   coredns-66c464876b-sr4fw                 1/1     Running   0          2m42s
kube-system   local-path-provisioner-7ff9579c6-5wpnq   1/1     Running   0          2m42s
kube-system   metrics-server-7b4f8b595-rmwl5           1/1     Running   0          2m42s
```

Forwarding ports to specific nodes in the cluster works the same as `k3d`

```bash
$ k3p install package.tar --docker \
    --publish 8080:80@loadbalancer \  # Forward 8080 on the local machine to 80 on the LoadBalancer
    --publish 8081:80@server[0]       # Forward 8081 on the local machine to 80 on the first server instance

2020/12/14 10:29:10  [INFO]     Loading the archive
2020/12/14 10:29:10  [INFO]     Creating docker network k3p-docker
2020/12/14 10:29:10  [INFO]     Creating docker volume k3p-docker-server-0
2020/12/14 10:29:10  [INFO]     Copying the archive to the rancher installation directory
2020/12/14 10:29:10  [INFO]     Installing binaries to /usr/local/bin
2020/12/14 10:29:10  [INFO]     Installing scripts to /usr/local/bin/k3p-scripts
2020/12/14 10:29:10  [INFO]     Installing manifests to /var/lib/rancher/k3s/server/manifests
2020/12/14 10:29:11  [INFO]     Running k3s installation script
2020/12/14 10:29:11  [INFO]     Starting k3s docker node k3p-docker-server-0
2020/12/14 10:29:11  [INFO]     Starting k3s docker node k3p-docker-serverlb
2020/12/14 10:29:12  [INFO]     The cluster has been installed
2020/12/14 10:29:12  [INFO]     You can retrieve the kubeconfig by running `docker cp k3p-docker-server-0:/etc/rancher/k3s/k3s.yaml ./kubeconfig.yaml`

$ docker ps
CONTAINER ID   IMAGE                      COMMAND                  CREATED         STATUS         PORTS                                          NAMES
b90d5ac9107a   rancher/k3d-proxy:latest   "/bin/sh -c nginx-pr…"   7 seconds ago   Up 5 seconds   0.0.0.0:6443->6443/tcp, 0.0.0.0:8080->80/tcp   k3p-docker-serverlb
f9d226f0a17b   rancher/k3s:v1.19.4-k3s1   "/bin/k3s server --t…"   7 seconds ago   Up 6 seconds   0.0.0.0:8081->80/tcp                           k3p-docker-server-0
```