## k3p token get

Retrieve a k3s token

### Synopsis


Retrieves the token for joining either a new "agent" or "server" to the cluster.

The "agent" token can be retrieved from any of the server instances, while the "server" token
can only be retrieved on the server where "k3p install" was run with "--init-ha".


```
k3p token get TOKEN_TYPE [flags]
```

### Options

```
  -h, --help   help for get
```

### Options inherited from parent commands

```
      --cache-dir string   Override the default location for cached k3s assets (default "/home/<user>/.k3p/cache")
      --tmp-dir string     Override the default tmp directory (default "/tmp")
  -v, --verbose            Enable verbose logging
```

### SEE ALSO

* [k3p token](k3p_token.md)	 - Token retrieval and generation commands

