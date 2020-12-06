## k3p install

Install the given package to the system (requires root)

### Synopsis


The install command can be used to distribute a package built with "k3p build".

The command takes a single argument (with optional flags) of the filesystem path or web URL
where the package resides. Additional flags provide the ability to initialize clustering (HA),
join existing servers, or pass custom arguments to the k3s agent/server processes.

Example

	$> k3p install /path/on/filesystem.tar
	$> k3p install https://example.com/package.tar

You can also direct the installation at a remote system over SSH via the --host flag.

    $> k3p install package.tar --host 192.168.1.100 [SSH_FLAGS]

See the help below for additional information on available flags.


```
k3p install PACKAGE [flags]
```

### Options

```
      --accept-eula              Automatically accept any EULA included with the package
  -h, --help                     help for install
  -H, --host string              The IP or DNS name of a remote host to perform the installation against
      --init-ha                  When set, this server will run with the --cluster-init flag to enable clustering, 
                                 and a token will be generated for adding additional servers to the cluster with 
                                 "--join-role server". You may optionally use the --join-token flag to provide a 
                                 pre-generated one.
  -j, --join string              When installing an agent instance, the address of the server to join (e.g. https://myserver:6443)
  -r, --join-role string         Specify whether to join the cluster as a "server" or "agent" (default "agent")
  -t, --join-token string        When installing an additional agent or server instance, the node token to use.
                                 
                                 For new agents, this can be retrieved with "k3p token get agent" or in 
                                 "/var/lib/rancher/k3s/server/node-token" on any of the server instances.
                                 For new servers, this value was either provided to or generated by 
                                 "k3s install --init-ha" and can be retrieved from that server with 
                                 "k3p token get server". When used with --init-ha, the provided token will 
                                 be used for registering new servers, instead of one being generated.
      --k3s-exec string          Extra arguments to pass to the k3s server or agent process, for more details see:
                                 https://rancher.com/docs/k3s/latest/en/installation/install-options/server-config
                                 
      --kubeconfig-mode string   The mode to set on the k3s kubeconfig. Default is to only allow root access
  -n, --node-name string         An optional name to give this node in the cluster
  -k, --private-key string       The path to a private key to use when authenticating against the remote host, 
                                 if not provided you will be prompted for a password (default "/home/<user>/.ssh/id_rsa")
      --resolv-conf string       The path of a resolv-conf file to use when configuring DNS in the cluster.
                                 When used with the --host flag, the path must reside on the remote system (this will change in the future).
  -p, --ssh-port int             The port to use when connecting to the remote host over SSH (default 22)
  -u, --ssh-user string          The username to use when authenticating against the remote host (default "<user>")
```

### Options inherited from parent commands

```
      --cache-dir string   Override the default location for cached k3s assets (default "/home/<user>/.k3p/cache")
      --tmp-dir string     Override the default tmp directory (default "/tmp")
  -v, --verbose            Enable verbose logging
```

### SEE ALSO

* [k3p](k3p.md)	 - k3p is a k3s packaging and delivery utility
