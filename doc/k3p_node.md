## k3p node

Node management commands

### Options

```
  -h, --help                 help for node
  -L, --leader string        The IP address or DNS name of the leader of the cluster.
                             
                             When left unset, the machine running k3p is assumed to be the leader of the cluster. Otherwise,
                             the provided host is remoted into, with the same connection options as for the new node in case 
                             of an add, to retrieve the installation manifest.
                             
  -k, --private-key string   A private key to use for SSH authentication, if not provided you will be prompted for a password (default "/home/<user>/.ssh/id_rsa")
  -p, --ssh-port int         The port to use when connecting to the remote instance over SSH (default 22)
  -u, --ssh-user string      The remote user to use for SSH authentication (default "<user>")
```

### Options inherited from parent commands

```
      --cache-dir string   Override the default location for cached k3s assets (default "/home/<user>/.k3p/cache")
      --tmp-dir string     Override the default tmp directory (default "/tmp")
  -v, --verbose            Enable verbose logging
```

### SEE ALSO

* [k3p](k3p.md)	 - k3p is a k3s packaging and delivery utility
* [k3p node add](k3p_node_add.md)	 - Add a new node to the cluster
* [k3p node remove](k3p_node_remove.md)	 - Remove a node from the cluster by name or IP

