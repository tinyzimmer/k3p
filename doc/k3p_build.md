## k3p build

Build an embedded k3s distribution package

```
k3p build [flags]
```

### Options

```
  -a, --arch string        The architecture to package the distribution for. Only (amd64, arm, and arm64 are supported) (default "amd64")
  -c, --channel string     The release channel to retrieve the version of k3s from (default "stable")
  -E, --eula string        A file containing an End User License Agreement to display to the user upon installing the package
  -e, --exclude strings    Directories to exclude when reading the manifest directory
  -H, --helm-args string   Arguments to pass to the 'helm template' command when searching for images
  -h, --help               help for build
  -i, --images string      A file containing a list of extra images to bundle with the archive
  -m, --manifests string   The directory to scan for kubernetes manifests and charts, defaults to the current directory (default "/home/aizimmerman/devel/k3p")
  -N, --no-cache           Disable the use of the local cache when downloading assets.
  -o, --output string      The file to save the distribution package to (default "/home/aizimmerman/devel/k3p/package.tar")
  -V, --version string     A specific k3s version to bundle with the package, overrides --channel (default "latest")
```

### Options inherited from parent commands

```
      --cache-dir string   Override the default location for cached k3s assets (default "/home/aizimmerman/.k3p/cache")
      --tmp-dir string     Override the default tmp directory (default "/tmp")
  -v, --verbose            Enable verbose logging
```

### SEE ALSO

* [k3p](k3p.md)	 - k3p is a k3s packaging and delivery utility

