## k3p node add

Add a new node to the cluster

```
k3p node add NODE [flags]
```

### Options

```
  -h, --help                 help for add
  -L, --leader string        The IP address or DNS name of the leader of the cluster.
                             
                             When left unset, the machine running k3p is assumed to be the leader of the cluster. Otherwise,
                             the provided host is remoted into, with the same connection options as for the new node, to retrieve
                             the installation manifest.
                             
  -r, --node-role string     Whether to join the instance as a 'server' or 'agent' (default "agent")
  -k, --private-key string   A private key to use for SSH authentication, if not provided you will be prompted for a password (default "/home/aizimmerman/.ssh/id_rsa")
  -p, --ssh-port int         The port to use when connecting to the remote instance over SSH (default 22)
  -u, --ssh-user string      The remote user to use for SSH authentication (default "aizimmerman")
```

### Options inherited from parent commands

```
      --cache-dir string   Override the default location for cached k3s assets (default "/home/aizimmerman/.k3p/cache")
      --tmp-dir string     Override the default tmp directory (default "/tmp")
  -v, --verbose            Enable verbose logging
```

### SEE ALSO

* [k3p node](k3p_node.md)	 - Node management commands

