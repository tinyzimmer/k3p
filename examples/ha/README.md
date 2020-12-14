## HA Deployments

In terms of the contents of the packag we again use a simple `whoami` example, except with this time specifying pod anti-affinity to ensure pods are not co-located on the same node.
K3p can be used to add new nodes to the cluster either locally via the `install` command, or remotely via the `node add` command.

### Creating the Initial Node

With the experimental k3s embedded etcd HA, one node has to be started with the `--cluster-init` flag, and then additional control-plane instances can be added through joining the initial node.

With the package in this directory already built, SSH in to your first host and run `k3p install` with the `--init-ha` flag. (This can also be done remotely with the `--host` flag).

```bash
[core@coreos1 ~]$ sudo k3p install package.tar --init-ha

2020/12/14 12:52:47  [INFO]     Loading the archive
2020/12/14 12:52:48  [INFO]     Copying the archive to the rancher installation directory
2020/12/14 12:52:49  [INFO]     Generating a node token for additional control-plane instances
2020/12/14 12:52:49  [INFO]     Installing binaries to /usr/local/bin
2020/12/14 12:52:49  [INFO]     Installing scripts to /usr/local/bin/k3p-scripts
2020/12/14 12:52:49  [INFO]     Installing images to /var/lib/rancher/k3s/agent/images
2020/12/14 12:52:50  [INFO]     Installing manifests to /var/lib/rancher/k3s/server/manifests
2020/12/14 12:52:50  [INFO]     Running k3s installation script
2020/12/14 12:52:50  [K3S]      [INFO]  Skipping k3s download and verify
2020/12/14 12:52:50  [K3S]      [INFO]  Skipping installation of SELinux RPM
2020/12/14 12:52:50  [K3S]      [INFO]  Creating /usr/local/bin/kubectl symlink to k3s
2020/12/14 12:52:50  [K3S]      [INFO]  Creating /usr/local/bin/crictl symlink to k3s
2020/12/14 12:52:50  [K3S]      [INFO]  Skipping /usr/local/bin/ctr symlink to k3s, command exists in PATH at /usr/bin/ctr
2020/12/14 12:52:50  [K3S]      [INFO]  Creating killall script /usr/local/bin/k3s-killall.sh
2020/12/14 12:52:50  [K3S]      [INFO]  Creating uninstall script /usr/local/bin/k3s-uninstall.sh
2020/12/14 12:52:50  [K3S]      [INFO]  env: Creating environment file /etc/systemd/system/k3s.service.env
2020/12/14 12:52:50  [K3S]      [INFO]  systemd: Creating service file /etc/systemd/system/k3s.service
2020/12/14 12:52:50  [K3S]      [INFO]  systemd: Enabling k3s unit
2020/12/14 12:52:50  [K3S]      Created symlink /etc/systemd/system/multi-user.target.wants/k3s.service → /etc/systemd/system/k3s.service.
2020/12/14 12:52:51  [K3S]      [INFO]  systemd: Starting k3s
2020/12/14 12:52:59  [INFO]     The cluster has been installed
2020/12/14 12:52:59  [INFO]     You can view the cluster by running `k3s kubectl cluster-info`

# A token was generated for joining new control-plane instances during the install.
# A pre-generated one can also be used. To retrieve the generated one you can run
[core@coreos1 ~]$ sudo k3p token get server
fFUiC96GBQ69XgENdvhabseBd53vSUVWuYhrLJKVRX08a3M9RA8qSYypBxLMX0iCEPBnWl6BmZ6WKIw4pAhtbQYhMWveiGI3YbkGMkwJQnTfuTnkBzzMIvsitvBiwqg3

# So far we have a single node, and we are unable to schedule two of our pods
[core@coreos1 ~]$ sudo k3s kubectl get node
NAME      STATUS   ROLES         AGE   VERSION
coreos1   Ready    etcd,master   68s   v1.19.4+k3s1

[core@coreos1 ~]$ sudo k3s kubectl get pod
NAME                      READY   STATUS    RESTARTS   AGE
whoami-5f47859667-87l9q   1/1     Running   0          63s
whoami-5f47859667-l77k6   0/1     Pending   0          63s
whoami-5f47859667-n8jvh   0/1     Pending   0          63s
```

To join a second and third instance to the cluster there are two (actually three) ways we can do this. The first way is to install the package again to the other instances, using the `--join` flag to signal joining an existing cluster.

```bash
[core@coreos2 ~]$ sudo k3p install package.tar \
    --join https://172.18.64.84:6443 \  # The IP and API port of the first instance
    --join-role server \                # Join as a server instance (the default option is as an agent and uses a different token)
    --join-token fFUiC96GBQ69XgENdvhabseBd53vSUVWuYhrLJKVRX08a3M9RA8qSYypBxLMX0iCEPBnWl6BmZ6WKIw4pAhtbQYhMWveiGI3YbkGMkwJQnTfuTnkBzzMIvsitvBiwqg3

# ...
# ...
```

You can also use `k3p node add` from the initial node to bring in new instances using SSH. If you have public key authentication setup you can use that, otherwise it will prompt for a password. 

You can also do this from a remote instance with the `--leader` flag assuming it uses the same SSH credentials as the new node you are adding.

```bash
[core@coreos1 ~]$ sudo k3p node add 172.18.64.91 \  # The remote address of the node
    --ssh-user core \                               # The user to use for SSH
    --private-key ~/.ssh/id_rsa \                   # The SSH private key (or omit to be prompted for a password)
    --node-role server                              # Join as a server

# ...
# ...
```

Once that is done you will have a highly available cluster and deployment

```bash
[core@coreos1 ~]$ sudo k3s kubectl get node
NAME      STATUS   ROLES         AGE     VERSION
coreos1   Ready    etcd,master   11m     v1.19.4+k3s1
coreos2   Ready    etcd,master   6m20s   v1.19.4+k3s1
coreos3   Ready    etcd,master   2m57s   v1.19.4+k3s1

[core@coreos1 ~]$ sudo k3s kubectl get pod
NAME                      READY   STATUS    RESTARTS   AGE
whoami-5f47859667-87l9q   1/1     Running   0          11m
whoami-5f47859667-l77k6   1/1     Running   0          11m
whoami-5f47859667-n8jvh   1/1     Running   0          11m
```